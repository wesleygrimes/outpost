package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wesleygrimes/outpost/internal/config"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/ui"
)

// Version is set by ldflags at build time.
var Version = "dev" //nolint:gochecknoglobals // injected from main via ldflags

// Doctor runs client-side health checks.
func Doctor() error {
	cl := ui.NewChecklist("Client Health")

	configPath := config.ClientConfigPath()
	cfg, err := config.LoadClient()
	if err != nil {
		cl.Fail("Config " + configPath)
		cl.Close()
		return fmt.Errorf("config: %w", err)
	}
	cl.Success("Config loaded (" + configPath + ")")

	client, err := grpcclient.Load()
	if err != nil {
		cl.Fail("Server " + cfg.Server + " unreachable")
		cl.Close()
		return nil
	}
	defer logClose(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.HealthCheck(ctx); err != nil {
		cl.Fail("Server " + cfg.Server + " unreachable")
	} else {
		cl.Success("Server reachable (" + cfg.Server + ")")
	}

	cl.Success("Outpost CLI " + Version)
	cl.Success("Skills " + countSkills())
	cl.Close()
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
