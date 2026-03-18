// Package cmd implements the outpost CLI subcommands.
package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
)

// Setup configures a new Outpost server with TLS certs, config, and systemd.
func Setup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	base := filepath.Join(home, ".outpost")
	tlsDir := filepath.Join(base, "tls")
	binDir := filepath.Join(base, "bin")
	runsDir := filepath.Join(base, "runs")

	fmt.Fprintln(os.Stderr, "Creating directories...")
	for _, d := range []string{base, tlsDir, binDir, runsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	fmt.Fprintln(os.Stderr, "Generating config with token...")
	cfg := config.Default()
	cfg.TLSCert = filepath.Join(tlsDir, "server.pem")
	cfg.TLSKey = filepath.Join(tlsDir, "server-key.pem")
	cfg.TLSCA = filepath.Join(tlsDir, "ca.pem")
	if err := cfg.SaveTo(filepath.Join(base, "config.yaml")); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Generating TLS certificates...")
	if err := generateTLSCerts(tlsDir); err != nil {
		return fmt.Errorf("generate TLS: %w", err)
	}

	// Copy binary.
	exe, err := os.Executable()
	if err == nil {
		dest := filepath.Join(binDir, "outpost")
		data, readErr := os.ReadFile(exe)
		if readErr == nil {
			_ = os.WriteFile(dest, data, 0o755)
		}
	}

	fmt.Fprintln(os.Stderr, "Checking Claude Code installation...")
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Fprintln(os.Stderr, "  WARNING: 'claude' not found in PATH")
	} else {
		fmt.Fprintln(os.Stderr, "  Claude Code found")
	}

	if runtime.GOOS == "linux" {
		installSystemd(base)
	}

	hostname, _ := os.Hostname()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "=== Outpost Setup Complete ===")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  Token:   %s\n", cfg.Token)
	fmt.Fprintf(os.Stderr, "  Address: 0.0.0.0:%d\n", cfg.Port)
	fmt.Fprintf(os.Stderr, "  CA cert: %s\n", filepath.Join(tlsDir, "ca.pem"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  On your laptop:")
	fmt.Fprintf(os.Stderr, "    scp %s:~/.outpost/tls/ca.pem /tmp/outpost-ca.pem\n", hostname)
	fmt.Fprintf(os.Stderr, "    outpost login %s:%d %s --ca-cert /tmp/outpost-ca.pem\n", hostname, cfg.Port, cfg.Token)

	return nil
}

func generateTLSCerts(tlsDir string) error {
	// Generate CA key.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Outpost CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA cert: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	// Generate server key.
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}

	sans := collectSANs()

	serverTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Outpost Server"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     sans.dns,
		IPAddresses:  sans.ips,
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create server cert: %w", err)
	}

	// Write PEM files.
	if err := writePEM(filepath.Join(tlsDir, "ca.pem"), "CERTIFICATE", caCertDER); err != nil {
		return err
	}
	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}
	if err := writePEM(filepath.Join(tlsDir, "ca-key.pem"), "EC PRIVATE KEY", caKeyDER); err != nil {
		return err
	}
	if err := writePEM(filepath.Join(tlsDir, "server.pem"), "CERTIFICATE", serverCertDER); err != nil {
		return err
	}
	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	return writePEM(filepath.Join(tlsDir, "server-key.pem"), "EC PRIVATE KEY", serverKeyDER)
}

type sanSet struct {
	dns []string
	ips []net.IP
}

func collectSANs() sanSet {
	s := sanSet{
		dns: []string{"localhost"},
	}

	if hostname, err := os.Hostname(); err == nil {
		s.dns = append(s.dns, hostname)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return s
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				s.ips = append(s.ips, ipNet.IP)
			}
		}
	}

	return s
}

func writePEM(path, blockType string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer logClose(f)

	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}

func installSystemd(base string) {
	unit := fmt.Sprintf(`[Unit]
Description=Outpost Server
After=network.target

[Service]
ExecStart=%s/bin/outpost serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, base)

	unitPath := "/etc/systemd/system/outpost.service"
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "  Note: could not write systemd unit: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "  Systemd unit installed: %s\n", unitPath)
}
