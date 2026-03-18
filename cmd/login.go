package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Login connects to an Outpost server and saves credentials.
func Login(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: outpost login <host:port> <token> [--ca-cert <path>]")
	}

	target := args[0]
	token := args[1]

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	var caCertPath string
	for i := 2; i < len(args)-1; i++ {
		if args[i] == "--ca-cert" {
			caCertPath = args[i+1]
			break
		}
	}

	if err := os.WriteFile(filepath.Join(home, ".outpost-url"), []byte(target), 0o600); err != nil {
		return fmt.Errorf("write url: %w", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".outpost-token"), []byte(token), 0o600); err != nil {
		return fmt.Errorf("write token: %w", err)
	}

	if caCertPath != "" {
		data, err := os.ReadFile(caCertPath)
		if err != nil {
			return fmt.Errorf("read CA cert: %w", err)
		}
		if err := os.WriteFile(filepath.Join(home, ".outpost-ca.pem"), data, 0o600); err != nil {
			return fmt.Errorf("write CA cert: %w", err)
		}
	}

	fmt.Fprint(os.Stderr, "Verifying connection... ")
	client, err := grpcclient.Load()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer logClose(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, status)
	fmt.Fprintln(os.Stderr, "Credentials saved.")

	return nil
}
