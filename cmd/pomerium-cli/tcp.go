package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/pomerium/pomerium/internal/log"
	"github.com/pomerium/pomerium/internal/tcptunnel"
)

var tcpCmd = &cobra.Command{
	Use: "tcp",
	RunE: func(cmd *cobra.Command, args []string) error {
		l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
		log.SetLogger(&l)

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-c
			cancel()
		}()

		err := tcptunnel.New().RunListener(ctx, "127.0.0.1:0")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tcpCmd)
}
