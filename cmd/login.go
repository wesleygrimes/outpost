package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/wesleygrimes/outpost/internal/config"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/ui"
)

// Login connects to an Outpost server and saves credentials.
func Login(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: outpost login <host:port> <token> [--ca-cert <path|host:path>]")
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

	clientCfg := &config.ClientConfig{
		Server: target,
		Token:  token,
	}

	// If --ca-cert explicitly provided, use it.
	if caCertPath != "" {
		destPath, err := fetchCACert(caCertPath)
		if err != nil {
			return err
		}
		clientCfg.CACert = destPath
		return saveAndVerify(clientCfg)
	}

	// Try system TLS first (works behind Traefik with real certs).
	ui.Errf("Connecting... ")
	if err := tryHealthCheck(target, token, nil); err == nil {
		return saveAndVerify(clientCfg)
	}

	// System TLS failed. Try TOFU: connect with InsecureSkipVerify,
	// grab the server's certificate, and pin it (like SSH does on first connect).
	caCert, err := fetchServerCert(target)
	if err != nil {
		return fmt.Errorf("could not connect: system TLS failed and direct TLS failed: %w", err)
	}

	// Save the server's CA cert and use it.
	destPath, err := saveCACert(caCert)
	if err != nil {
		return err
	}
	clientCfg.CACert = destPath

	return saveAndVerify(clientCfg)
}

func saveAndVerify(cfg *config.ClientConfig) error {
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

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
	ui.Errf("Connected: %s\n", status)
	ui.Errf("Credentials saved to %s\n", config.ClientConfigPath())
	return nil
}

// tryHealthCheck attempts a HealthCheck with the given CA pool (nil = system roots).
func tryHealthCheck(target, token string, pool *x509.CertPool) error {
	opt := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}))
	client, err := grpcclient.New(target, token, opt)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.HealthCheck(ctx)
	return err
}

// fetchServerCert connects with InsecureSkipVerify and returns the server's
// leaf certificate in PEM form. This is the TOFU (Trust On First Use) model.
func fetchServerCert(target string) ([]byte, error) {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp", target,
		&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // TOFU: we verify on subsequent connections
			MinVersion:         tls.VersionTLS12,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}
	defer func() { _ = conn.Close() }()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, errors.New("server presented no certificates")
	}

	// Use the last cert in the chain (the CA / self-signed root).
	leaf := certs[len(certs)-1]
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: leaf.Raw,
	}), nil
}

// saveCACert writes PEM data to ~/.config/outpost/ca.pem.
func saveCACert(pemData []byte) (string, error) {
	destDir := config.ClientConfigDir()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	destPath := filepath.Join(destDir, "ca.pem")
	if err := os.WriteFile(destPath, pemData, 0o600); err != nil {
		return "", fmt.Errorf("write CA cert: %w", err)
	}
	return destPath, nil
}

// fetchCACert handles both local paths and remote scp paths (host:path).
func fetchCACert(src string) (string, error) {
	destDir := config.ClientConfigDir()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	destPath := filepath.Join(destDir, "ca.pem")

	if strings.Contains(src, ":") {
		out, err := exec.Command("scp", src, destPath).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("scp CA cert: %w: %s", err, out)
		}
		return destPath, nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read CA cert: %w", err)
	}
	if err := os.WriteFile(destPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write CA cert: %w", err)
	}
	return destPath, nil
}
