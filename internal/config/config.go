// Package config handles Outpost server configuration.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Default port, concurrency, and token size constants.
const (
	DefaultPort              = 7600
	DefaultMaxConcurrentRuns = 3
	TokenBytes               = 32
)

// ServerConfig holds the Outpost server settings.
type ServerConfig struct {
	Port              int    `yaml:"port"`
	Token             string `yaml:"token"`
	MaxConcurrentRuns int    `yaml:"max_concurrent_runs"`
	SSHUser           string `yaml:"ssh_user"`
	TLSCert           string `yaml:"tls_cert"`
	TLSKey            string `yaml:"tls_key"`
	TLSCA             string `yaml:"tls_ca"`
}

// Default returns a ServerConfig with generated token and defaults.
func Default() *ServerConfig {
	token := make([]byte, TokenBytes)
	if _, err := rand.Read(token); err != nil {
		panic("failed to generate token: " + err.Error())
	}

	return &ServerConfig{
		Port:              DefaultPort,
		Token:             hex.EncodeToString(token),
		MaxConcurrentRuns: DefaultMaxConcurrentRuns,
	}
}

// Load reads the config from the default path ~/.outpost/config.yaml.
func Load() (*ServerConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	return LoadFrom(filepath.Join(home, ".outpost", "config.yaml"))
}

// LoadFrom reads and parses a config file, applying env overrides.
func LoadFrom(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg ServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if v := os.Getenv("OUTPOST_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("OUTPOST_TOKEN"); v != "" {
		cfg.Token = v
	}

	return &cfg, nil
}

// Save writes the config to the default path.
func (c *ServerConfig) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	return c.SaveTo(filepath.Join(home, ".outpost", "config.yaml"))
}

// SaveTo writes the config to a specific path.
func (c *ServerConfig) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// ErrNotFound indicates a missing config file.
var ErrNotFound = errors.New("config not found")

// RunsDir returns the path to the runs directory (~/.outpost/runs).
func RunsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/outpost-runs"
	}
	return filepath.Join(home, ".outpost", "runs")
}
