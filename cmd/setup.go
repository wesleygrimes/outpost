package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/wesgrimes/outpost/internal/config"
)

const systemdUnit = `[Unit]
Description=Outpost - Remote Claude Code Runner
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s serve
Restart=on-failure
RestartSec=5
WorkingDirectory=%s
Environment=PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
Environment=HOME=%s

[Install]
WantedBy=multi-user.target
`

// Setup runs the interactive setup wizard.
func Setup() {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("getting home directory: %v", err)
	}

	outpostDir := filepath.Join(home, ".outpost")
	runsDir := filepath.Join(outpostDir, "runs")
	binDir := filepath.Join(outpostDir, "bin")
	configPath := config.DefaultPath()

	// 1. Create directories.
	fmt.Println("Creating directories...")

	for _, dir := range []string{outpostDir, runsDir, binDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			fatalf("creating %s: %v", dir, err)
		}
	}

	// 2. Copy binary.
	exe, err := os.Executable()
	if err != nil {
		fatalf("finding executable: %v", err)
	}

	destBin := filepath.Join(binDir, "outpost")
	if exe != destBin {
		fmt.Printf("Copying binary to %s...\n", destBin)

		input, err := os.ReadFile(exe)
		if err != nil {
			fatalf("reading binary: %v", err)
		}

		if err := os.WriteFile(destBin, input, 0o755); err != nil {
			fatalf("writing binary: %v", err)
		}
	}

	// 3. Generate config.
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s, skipping generation.\n", configPath)
	} else {
		fmt.Println("Generating config...")

		cfg, err := config.Default()
		if err != nil {
			fatalf("generating default config: %v", err)
		}

		if err := cfg.SaveTo(configPath); err != nil {
			fatalf("saving config: %v", err)
		}

		fmt.Printf("Config written to %s\n", configPath)
		fmt.Printf("Bearer token: %s\n", cfg.Server.Token)
	}

	// 4. Install systemd unit (Linux only).
	if runtime.GOOS == "linux" {
		installSystemd(home, destBin, outpostDir)
	} else {
		fmt.Println("Skipping systemd setup (not Linux).")
	}

	// 5. Check Claude Code.
	checkClaude()

	// 6. Print summary.
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		fatalf("loading config: %v", err)
	}

	printSummary(cfg, home)
}

func installSystemd(home, binPath, workDir string) {
	user := os.Getenv("USER")
	if user == "" {
		user = "wes"
	}

	unitContent := fmt.Sprintf(systemdUnit, user, binPath, workDir, home)
	unitPath := "/etc/systemd/system/outpost.service"

	fmt.Printf("Installing systemd unit to %s...\n", unitPath)

	if err := os.WriteFile(unitPath, []byte(unitContent), 0o644); err != nil {
		fmt.Printf("Warning: could not write systemd unit (run as root?): %v\n", err)
		fmt.Printf("Manual install:\n%s\n", unitContent)

		return
	}

	// Enable the service.
	if out, err := exec.Command("systemctl", "enable", "outpost").CombinedOutput(); err != nil {
		fmt.Printf("Warning: systemctl enable failed: %v\n%s\n", err, out)
	} else {
		fmt.Println("Service enabled.")
	}
}

func checkClaude() {
	fmt.Println("Checking Claude Code installation...")

	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Println("Warning: 'claude' not found in PATH.")
		fmt.Println("Install: npm install -g @anthropic-ai/claude-code")
		fmt.Println("Then run 'claude' to authenticate with your Max subscription.")

		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	claudeDir := filepath.Join(home, ".claude")

	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		fmt.Println("Warning: ~/.claude not found. Run 'claude' to authenticate.")
		return
	}

	fmt.Println("Claude Code is installed and configured.")
}

func printSummary(cfg *config.Config, home string) {
	fmt.Println()
	fmt.Println("=== Outpost Setup Complete ===")
	fmt.Println()
	fmt.Printf("Token: %s\n", cfg.Server.Token)
	fmt.Printf("Port:  %d\n", cfg.Server.Port)
	fmt.Println()
	fmt.Println("On your laptop:")
	fmt.Println()
	fmt.Printf("  outpost login http://<this-host>:%d %s\n", cfg.Server.Port, cfg.Server.Token)
	fmt.Println()
	fmt.Println("SSH config (~/.ssh/config):")
	fmt.Println()
	fmt.Println("  Host outpost")
	fmt.Println("    HostName <this-host-ip>")
	fmt.Printf("    User %s\n", os.Getenv("USER"))
	fmt.Println()

	if runtime.GOOS == "linux" {
		fmt.Println("Start the daemon:")
		fmt.Println("  sudo systemctl start outpost")
		fmt.Println()
	} else {
		fmt.Printf("Start manually:\n  %s/.outpost/bin/outpost serve\n\n", home)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
