package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ClientConfig holds the client-side connection settings.
type ClientConfig struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
	CACert string `yaml:"ca_cert,omitempty"`
}

// ClientConfigDir returns the client config directory (~/.config/outpost/).
func ClientConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "outpost")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "outpost-config")
	}
	return filepath.Join(home, ".config", "outpost")
}

// ClientConfigPath returns the full path to config.yaml.
func ClientConfigPath() string {
	return filepath.Join(ClientConfigDir(), "config.yaml")
}

// LoadClient reads the client config from ~/.config/outpost/config.yaml,
// auto-migrating old dotfiles if needed.
func LoadClient() (*ClientConfig, error) {
	path := ClientConfigPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if migrated, mErr := MigrateOldDotfiles(); mErr != nil {
			return nil, fmt.Errorf("migrate old config: %w", mErr)
		} else if !migrated {
			return nil, fmt.Errorf("config not found at %s (run 'outpost login' first)", path)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg ClientConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Server == "" || cfg.Token == "" {
		return nil, fmt.Errorf("config at %s is missing server or token", path)
	}

	return &cfg, nil
}

// Save writes the client config to ~/.config/outpost/config.yaml.
func (c *ClientConfig) Save() error {
	return c.SaveTo(ClientConfigPath())
}

// SaveTo writes the client config to a specific path.
func (c *ClientConfig) SaveTo(path string) error {
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
