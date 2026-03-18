package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Version is set by ldflags at build time.
var Version = "dev" //nolint:gochecknoglobals // injected from main via ldflags

// Doctor runs client-side health checks.
func Doctor() error {
	printBoxTop("Outpost Doctor", "")

	// Check config.
	configPath := config.ClientConfigPath()
	cfg, err := config.LoadClient()
	if err != nil {
		printFailItem("Config", configPath)
		printBoxBottom()
		return fmt.Errorf("config: %w", err)
	}
	printCheckItem("Config", configPath)

	// Check server reachable.
	client, err := grpcclient.Load()
	if err != nil {
		printFailItem("Server", cfg.Server+" unreachable")
		printBoxBottom()
		return nil
	}
	defer logClose(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.HealthCheck(ctx); err != nil {
		printFailItem("Server", cfg.Server+" unreachable")
	} else {
		printCheckItem("Server", cfg.Server+" reachable")
	}

	// CLI version.
	printCheckItem("CLI version", Version)

	// Skills check.
	skills := countSkills()
	printCheckItem("Skills", skills)

	printBoxBottom()
	return nil
}

func countSkills() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "unknown"
	}

	skillsDir := filepath.Join(home, ".claude", "commands")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		// Try global install path.
		out, err := exec.Command("claude", "skill", "list").CombinedOutput()
		if err != nil {
			return "0 found"
		}
		// Count lines.
		count := 0
		for _, b := range out {
			if b == '\n' {
				count++
			}
		}
		return fmt.Sprintf("%d found", count)
	}

	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return fmt.Sprintf("%d installed", count)
}
