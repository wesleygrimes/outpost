package runner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// ModeInteractive runs Claude Code in an attachable tmux session.
	ModeInteractive = "interactive"
	// ModeHeadless runs Claude Code non-interactively with --dangerously-skip-permissions.
	ModeHeadless = "headless"

	defaultMaxTurns  = 50
	runIDBytes       = 4 // 8 hex chars, ~4 billion combinations
	exitCodeFileName = ".outpost-exit-code"
	pollInterval     = 5 * time.Second
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SpawnConfig configures a new Claude Code session.
type SpawnConfig struct {
	RunID    string
	RepoDir  string
	PlanPath string
	LogPath  string
	Mode     string
	MaxTurns int
	OnExit   func(exitCode int)
}

// ValidateMode returns an error if mode is not a recognized value.
func ValidateMode(mode string) error {
	if mode != ModeInteractive && mode != ModeHeadless {
		return fmt.Errorf("mode must be %s or %s", ModeInteractive, ModeHeadless)
	}

	return nil
}

// Spawn runs Claude Code against the repo. Interactive mode uses a tmux session
// that can be attached via SSH. Headless mode runs directly in the background.
// OnExit is called in a goroutine when the process finishes.
func Spawn(cfg *SpawnConfig) error {
	claudeCmd := buildClaudeCmd(cfg)
	exitCodeFile := filepath.Join(cfg.RepoDir, "..", exitCodeFileName)

	// The wrapper script: run claude, capture output, write exit code.
	script := fmt.Sprintf(
		"cd %s && %s > %s 2>&1; echo $? > %s",
		cfg.RepoDir, claudeCmd, cfg.LogPath, exitCodeFile,
	)

	if cfg.Mode == ModeInteractive {
		return spawnInteractive(cfg, script, exitCodeFile)
	}

	return spawnHeadless(cfg, script, exitCodeFile)
}

func buildClaudeCmd(cfg *SpawnConfig) string {
	if cfg.Mode == ModeHeadless {
		maxTurns := cfg.MaxTurns
		if maxTurns == 0 {
			maxTurns = defaultMaxTurns
		}

		return fmt.Sprintf(
			"claude -p --dangerously-skip-permissions --max-turns %d \"$(cat %s)\"",
			maxTurns, cfg.PlanPath,
		)
	}

	return fmt.Sprintf(
		"claude \"Read the plan at %s and execute it fully. Do not ask clarifying questions.\"",
		cfg.PlanPath,
	)
}

func spawnHeadless(cfg *SpawnConfig, script, exitCodeFile string) error {
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = cfg.RepoDir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting headless session: %w", err)
	}

	go func() {
		_ = cmd.Wait()

		exitCode := readExitCode(exitCodeFile)
		if cfg.OnExit != nil {
			cfg.OnExit(exitCode)
		}
	}()

	return nil
}

func spawnInteractive(cfg *SpawnConfig, script, exitCodeFile string) error {
	// tmux creates a detached session that the user can attach to via SSH:
	//   ssh outpost -t 'tmux attach -t <run-id>'
	cmd := exec.Command(
		"tmux", "new-session", "-d", "-s", cfg.RunID,
		"bash", "-c", script,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("starting tmux session: %w", err)
	}

	// Monitor the tmux session in the background.
	go monitorTmuxSession(cfg, exitCodeFile)

	return nil
}

func monitorTmuxSession(cfg *SpawnConfig, exitCodeFile string) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Check if the tmux session still exists.
		err := exec.Command("tmux", "has-session", "-t", cfg.RunID).Run()
		if err != nil {
			// Session is gone; process finished.
			exitCode := readExitCode(exitCodeFile)
			if cfg.OnExit != nil {
				cfg.OnExit(exitCode)
			}

			return
		}
	}
}

// Kill terminates a run session (tmux or process).
func Kill(runID string) error {
	// Try tmux first.
	if err := exec.Command("tmux", "kill-session", "-t", runID).Run(); err == nil {
		return nil
	}

	// Fall back to pkill if it was a headless process.
	// The process doesn't have a session name, so this is best-effort.
	return nil
}

// GenerateRunID creates a run ID like "name-20260317-143022-a1b2c3d4".
// Name is sanitized to alphanumeric, hyphens, and underscores.
func GenerateRunID(name string) string {
	b := make([]byte, runIDBytes)
	_, _ = rand.Read(b)

	suffix := hex.EncodeToString(b)
	ts := time.Now().Format("20060102-150405")

	name = sanitizeName(name)

	return fmt.Sprintf("%s-%s-%s", name, ts, suffix)
}

func sanitizeName(name string) string {
	if name == "" {
		return "run"
	}

	if validNameRe.MatchString(name) {
		return name
	}

	// Strip invalid characters.
	var clean strings.Builder

	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			clean.WriteRune(r)
		}
	}

	if clean.Len() == 0 {
		return "run"
	}

	return clean.String()
}

func readExitCode(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 1
	}

	code, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 1
	}

	_ = os.Remove(path)

	return code
}
