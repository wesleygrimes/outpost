package runner

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	// ModeInteractive runs Claude Code in an attachable zellij session.
	ModeInteractive = "interactive"
	// ModeHeadless runs Claude Code non-interactively with --dangerously-skip-permissions.
	ModeHeadless = "headless"

	defaultMaxTurns  = 50
	runIDBytes       = 4 // 8 hex chars, ~4 billion combinations
	sessionPollSec   = 5
	sessionMaxAge    = 24 * time.Hour
	exitCodeFileName = ".outpost-exit-code"
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

// Spawn creates a zellij session and runs Claude Code inside it.
// The session is monitored in a background goroutine; OnExit is called when it finishes.
func Spawn(cfg *SpawnConfig) error {
	var claudeCmd string

	switch cfg.Mode {
	case ModeHeadless:
		maxTurns := cfg.MaxTurns
		if maxTurns == 0 {
			maxTurns = defaultMaxTurns
		}

		claudeCmd = fmt.Sprintf(
			"claude -p --dangerously-skip-permissions --max-turns %d \"$(cat %s)\"",
			maxTurns, cfg.PlanPath,
		)
	default:
		claudeCmd = fmt.Sprintf(
			"claude \"Read the plan at %s and execute it fully. Do not ask clarifying questions.\"",
			cfg.PlanPath,
		)
	}

	// Wrap command to capture exit code to a marker file, then tee output to log.
	exitCodeFile := filepath.Join(cfg.RepoDir, "..", exitCodeFileName)
	shellCmd := fmt.Sprintf(
		"cd %s && (%s 2>&1 | tee %s); echo $? > %s",
		cfg.RepoDir, claudeCmd, cfg.LogPath, exitCodeFile,
	)

	cmd := exec.Command(
		"zellij", "--session", cfg.RunID,
		"--",
		"bash", "-c", shellCmd,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting zellij session: %w", err)
	}

	go monitorProcess(cmd, cfg, exitCodeFile)

	return nil
}

// Kill terminates a zellij session.
func Kill(runID string) error {
	cmd := exec.Command("zellij", "kill-session", runID)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("killing session %s: %w\n%s", runID, err, out)
	}

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

func monitorProcess(cmd *exec.Cmd, cfg *SpawnConfig, exitCodeFile string) {
	_ = cmd.Wait()

	// Read exit code from marker file (captures the claude process exit code,
	// not the zellij wrapper exit code).
	exitCode := readExitCode(exitCodeFile)

	if cfg.OnExit != nil {
		cfg.OnExit(exitCode)
	}
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

	// Clean up marker file.
	_ = os.Remove(path)

	return code
}

// CheckZellijInstalled verifies that zellij is available in PATH.
func CheckZellijInstalled() error {
	_, err := exec.LookPath("zellij")
	if err != nil {
		return errors.New("zellij not found in PATH")
	}

	return nil
}
