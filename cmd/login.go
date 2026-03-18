package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Login connects to an Outpost server and saves credentials.
func Login(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: outpost login <host:port> <token> [--ca-cert <path>]")
	}

	target := args[0]
	token := args[1]

	var caCertPath string
	for i := 2; i < len(args)-1; i++ {
		if args[i] == "--ca-cert" {
			caCertPath = args[i+1]
			break
		}
	}

	// Build client config.
	clientCfg := &config.ClientConfig{
		Server: target,
		Token:  token,
	}

	// If --ca-cert provided, copy it to config dir.
	if caCertPath != "" {
		data, err := os.ReadFile(caCertPath)
		if err != nil {
			return fmt.Errorf("read CA cert: %w", err)
		}
		destDir := config.ClientConfigDir()
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		destPath := destDir + "/ca.pem"
		if err := os.WriteFile(destPath, data, 0o600); err != nil {
			return fmt.Errorf("write CA cert: %w", err)
		}
		clientCfg.CACert = destPath
	}

	if err := clientCfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
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
	fmt.Fprintf(os.Stderr, "Credentials saved to %s\n", config.ClientConfigPath())

	return nil
}
