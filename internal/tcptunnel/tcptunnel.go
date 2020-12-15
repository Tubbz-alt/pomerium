// Package tcptunnel contains an implementation of a TCP tunnel via HTTP Connect.
package tcptunnel

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pomerium/pomerium/internal/log"

	backoff "github.com/cenkalti/backoff/v4"
)

// A Tunnel represents a TCP tunnel over HTTP Connect.
type Tunnel struct {
	proxyHost string
	dstHost   string
	tlsConfig *tls.Config
}

// New creates a new Tunnel.
func New() *Tunnel {
	return &Tunnel{
		proxyHost: "redis.caleb-pc-linux.doxsey.net:443",
		dstHost:   "redis.caleb-pc-linux.doxsey.net:22",
		tlsConfig: new(tls.Config),
	}
}

// RunListener runs a network listener on the given address. For each
// incoming connection a new TCP tunnel is established via Run.
func (tun *Tunnel) RunListener(ctx context.Context, listenerAddress string) error {
	li, err := net.Listen("tcp", listenerAddress)
	if err != nil {
		return err
	}
	defer func() { _ = li.Close() }()
	log.Info().Msg("tcptunnel: listening on " + li.Addr().String())

	go func() {
		<-ctx.Done()
		_ = li.Close()
	}()

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0

	for {
		conn, err := li.Accept()
		if err != nil {
			// canceled, so ignore the error and return
			if ctx.Err() != nil {
				return nil
			}

			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				log.Warn().Err(err).Msg("tcptunnel: temporarily failed to accept local connection")
				select {
				case <-time.After(bo.NextBackOff()):
				case <-ctx.Done():
					return ctx.Err()
				}
				continue
			}
			return err
		}
		bo.Reset()

		go func() {
			defer func() { _ = conn.Close() }()

			err := tun.Run(ctx, conn)
			if err != nil {
				log.Error().Err(err).Msg("tcptunnel: error serving local connection")
			}
		}()
	}
}

// Run establishes a TCP tunnel via HTTP Connect and forwards all traffic from/to local.
func (tun *Tunnel) Run(ctx context.Context, local io.ReadWriter) error {
	log.Info().
		Str("dst", tun.dstHost).
		Str("proxy", tun.proxyHost).
		Bool("secure", tun.tlsConfig != nil).
		Msg("tcptunnel: opening connection")

	req := (&http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: tun.dstHost},
		Host:   tun.dstHost,
	}).WithContext(ctx)

	var remote net.Conn
	var err error
	if tun.tlsConfig != nil {
		remote, err = (&tls.Dialer{Config: tun.tlsConfig}).DialContext(ctx, "tcp", tun.proxyHost)
	} else {
		remote, err = (&net.Dialer{}).DialContext(ctx, "tcp", tun.proxyHost)
	}
	if err != nil {
		return fmt.Errorf("tcptunnel: failed to establish connection to proxy: %w", err)
	}
	defer func() {
		_ = remote.Close()
		log.Info().Msg("tcptunnel: connection closed")
	}()

	err = req.Write(remote)
	if err != nil {
		return err
	}

	br := bufio.NewReader(remote)
	res, err := http.ReadResponse(br, req)
	if err != nil {
		return fmt.Errorf("tcptunnel: failed to read HTTP response: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		return fmt.Errorf("tcptunnel: invalid http response code: %d", res.StatusCode)
	}

	log.Info().Msg("tcptunnel: connection established")

	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(remote, local)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(local, remote)
		errc <- err
	}()

	select {
	case err := <-errc:
		return fmt.Errorf("tcptunnel: %w", err)
	case <-ctx.Done():
		return nil
	}
}
