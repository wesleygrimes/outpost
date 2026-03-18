package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrom_ValidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `port: 8080
token: "mytoken123"
max_concurrent_runs: 5
tls_cert: /path/to/cert
tls_key: /path/to/key
tls_ca: /path/to/ca
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Token != "mytoken123" {
		t.Errorf("Token = %q, want %q", cfg.Token, "mytoken123")
	}
	if cfg.MaxConcurrentRuns != 5 {
		t.Errorf("MaxConcurrentRuns = %d, want 5", cfg.MaxConcurrentRuns)
	}
	if cfg.TLSCert != "/path/to/cert" {
		t.Errorf("TLSCert = %q", cfg.TLSCert)
	}
}

func TestLoadFrom_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("port: 7600\ntoken: filetoken\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OUTPOST_PORT", "9999")
	t.Setenv("OUTPOST_TOKEN", "envtoken")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Port != 9999 {
		t.Errorf("Port = %d, want 9999 (env override)", cfg.Port)
	}
	if cfg.Token != "envtoken" {
		t.Errorf("Token = %q, want %q (env override)", cfg.Token, "envtoken")
	}
}

func TestLoadFrom_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := LoadFrom("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFrom_InvalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSaveTo_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &ServerConfig{
		Port:              8080,
		Token:             "testtoken",
		MaxConcurrentRuns: 2,
		TLSCert:           "/cert",
		TLSKey:            "/key",
		TLSCA:             "/ca",
	}

	if err := original.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if loaded.Port != original.Port {
		t.Errorf("Port: got %d, want %d", loaded.Port, original.Port)
	}
	if loaded.Token != original.Token {
		t.Errorf("Token: got %q, want %q", loaded.Token, original.Token)
	}
	if loaded.MaxConcurrentRuns != original.MaxConcurrentRuns {
		t.Errorf("MaxConcurrentRuns: got %d, want %d", loaded.MaxConcurrentRuns, original.MaxConcurrentRuns)
	}
}

func TestDefault_TokenGeneration(t *testing.T) {
	t.Parallel()
	c1 := Default()
	c2 := Default()

	if len(c1.Token) != 64 {
		t.Errorf("token length = %d, want 64 hex chars", len(c1.Token))
	}
	if c1.Token == c2.Token {
		t.Error("two Default() calls produced the same token")
	}
	if c1.Port != DefaultPort {
		t.Errorf("Port = %d, want %d", c1.Port, DefaultPort)
	}
	if c1.MaxConcurrentRuns != DefaultMaxConcurrentRuns {
		t.Errorf("MaxConcurrentRuns = %d, want %d", c1.MaxConcurrentRuns, DefaultMaxConcurrentRuns)
	}
}
