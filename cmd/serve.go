package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcserver"
	"github.com/wesgrimes/outpost/internal/store"
)

// Serve loads config, creates the gRPC server, and handles signals.
func Serve() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st := store.New()
	srv, err := grpcserver.New(cfg, st)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\noutpost: shutting down...")
		srv.GracefulStop()
	}()

	return srv.ListenAndServe()
}
