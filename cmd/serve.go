package cmd

import (
	"fmt"
	"os"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/server"
	"github.com/wesgrimes/outpost/internal/store"
)

// Serve loads config and starts the HTTP server.
func Serve() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'outpost setup' first.")
		os.Exit(1)
	}

	st := store.New()
	srv := server.New(cfg, st)

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
