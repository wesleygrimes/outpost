package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	defaultPort          = 7600
	defaultMaxConcurrent = 3
	tokenBytes           = 32
)

// Config holds the full Outpost configuration.
type Config struct {
	Server ServerConfig `yaml:"server"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port              int    `yaml:"port"`
	Token             string `yaml:"token"`
	MaxConcurrentRuns int    `yaml:"max_concurrent_runs"`
}

// Default returns a Config with sensible defaults and a generated token.
func Default() (*Config, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port:              defaultPort,
			Token:             token,
			MaxConcurrentRuns: defaultMaxConcurrent,
		},
	}, nil
}

// DefaultPath returns ~/.outpost/config.yaml.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".outpost", "config.yaml")
	}

	return filepath.Join(home, ".outpost", "config.yaml")
}

// Load reads config from DefaultPath and applies env overrides.
func Load() (*Config, error) {
	return LoadFrom(DefaultPath())
}

// LoadFrom reads config from the given path and applies env overrides.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

// Save writes config to DefaultPath.
func (c *Config) Save() error {
	return c.SaveTo(DefaultPath())
}

// SaveTo writes config to the given path, creating parent directories as needed.
func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("OUTPOST_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}

	if v := os.Getenv("OUTPOST_TOKEN"); v != "" {
		cfg.Server.Token = v
	}
}

func generateToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
