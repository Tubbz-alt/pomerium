package authclient

import (
	"crypto/tls"
	"net/url"
)

type config struct {
	cache     Cache
	serverURL *url.URL
	tlsConfig *tls.Config
}

func getConfig(options ...Option) *config {

}

// An Option modifies the config.
type Option func(*config)

// WithCache returns an option to configure the cache.
func WithCache(cache Cache) Option {
	return func(cfg *config) {
		cfg.cache = cache
	}
}

// WithServerURL returns an option to configure the server url.
func WithServerURL(serverURL *url.URL) Option {
	return func(cfg *config) {
		cfg.serverURL = serverURL
	}
}

// WithTLSConfig returns an option to configure the tls config.
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(cfg *config) {
		cfg.tlsConfig = tlsConfig
	}
}
