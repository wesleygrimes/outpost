package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrateOldDotfiles checks for the old ~/.outpost-{url,token,ca.pem} files
// and migrates them to ~/.config/outpost/config.yaml. Returns true if migration occurred.
func MigrateOldDotfiles() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("home dir: %w", err)
	}

	urlPath := filepath.Join(home, ".outpost-url")
	tokenPath := filepath.Join(home, ".outpost-token")
	caPath := filepath.Join(home, ".outpost-ca.pem")

	urlData, urlErr := os.ReadFile(urlPath)
	tokenData, tokenErr := os.ReadFile(tokenPath)

	// No old dotfiles found means nothing to migrate (not an error).
	if urlErr != nil || tokenErr != nil {
		return false, nil //nolint:nilerr // missing dotfiles is expected, not an error
	}

	cfg := &ClientConfig{
		Server: strings.TrimSpace(string(urlData)),
		Token:  strings.TrimSpace(string(tokenData)),
	}

	if caData, err := os.ReadFile(caPath); err == nil {
		caDestDir := ClientConfigDir()
		caDestPath := filepath.Join(caDestDir, "ca.pem")
		if err := os.MkdirAll(caDestDir, 0o755); err != nil {
			return false, fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(caDestPath, caData, 0o600); err != nil {
			return false, fmt.Errorf("write ca cert: %w", err)
		}
		cfg.CACert = caDestPath
	}

	if err := cfg.Save(); err != nil {
		return false, fmt.Errorf("save migrated config: %w", err)
	}

	// Clean up old files (best-effort).
	_ = os.Remove(urlPath)
	_ = os.Remove(tokenPath)
	_ = os.Remove(caPath)

	fmt.Fprintf(os.Stderr, "Migrated old config to %s\n", ClientConfigPath())
	return true, nil
}
