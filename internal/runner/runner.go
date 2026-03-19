// Package runner spawns and manages Claude Code sessions.
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
	"sync"
	"syscall"
	"time"

	"github.com/wesgrimes/outpost/internal/store"
)

// Timing constants.
const (
	DefaultMaxTurns  = 50
	ExitCodeFileName = ".outpost-exit-code"
	pollInterval     = 5 * time.Second
	headlessStopWait = 5 * time.Second
)

// Registry tracks headless processes for stop/signal support.
type Registry struct {
	mu        sync.RWMutex
	processes map[string]*os.Process
}

// NewRegistry creates an empty process registry.
func NewRegistry() *Registry {
	return &Registry{processes: make(map[string]*os.Process)}
}

func (r *Registry) add(id string, p *os.Process) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processes[id] = p
}

func (r *Registry) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.processes, id)
}

func (r *Registry) get(id string) (*os.Process, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.processes[id]
	return p, ok
}

func (r *Registry) has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.processes[id]
	return ok
}

// SpawnConfig holds parameters for launching a session.
type SpawnConfig struct {
	RunID    string
	RepoDir  string
	PlanPath string
	LogPath  string
	Mode     store.Mode
	MaxTurns int
	OnExit   func(exitCode int)
	Registry *Registry
}

// BuildClaudeCmd constructs the claude CLI invocation string.
func BuildClaudeCmd(cfg *SpawnConfig) string {
	maxTurns := cfg.MaxTurns
	if maxTurns == 0 {
		maxTurns = DefaultMaxTurns
	}

	var args []string
	args = append(args, "claude")

	switch cfg.Mode {
	case store.ModeHeadless:
		args = append(args,
			"--print",
			"--permission-mode", "bypassPermissions",
			"--max-turns", strconv.Itoa(maxTurns),
		)
	case store.ModeInteractive, "":
		args = append(args,
			"--permission-mode", "bypassPermissions",
			"--max-turns", strconv.Itoa(maxTurns),
		)
	}

	return strings.Join(args, " ") + fmt.Sprintf(` < %q`, cfg.PlanPath)
}

func buildWrapperScript(cfg *SpawnConfig) string {
	exitCodePath := filepath.Join(filepath.Dir(cfg.RepoDir), ExitCodeFileName)

	switch cfg.Mode {
	case store.ModeInteractive:
		// Interactive: run directly in the tmux PTY so the TUI renders
		// and the pre-seeded workspace trust is respected.
		return fmt.Sprintf(
			"cd %q && %s; echo $? > %q",
			cfg.RepoDir,
			BuildClaudeCmd(cfg),
			exitCodePath,
		)
	case store.ModeHeadless:
		// Headless: redirect stdout/stderr to log file.
		return fmt.Sprintf(
			"cd %q && %s > %q 2>&1; echo $? > %q",
			cfg.RepoDir, BuildClaudeCmd(cfg), cfg.LogPath,
			exitCodePath,
		)
	}

	panic("unreachable: unknown mode " + string(cfg.Mode))
}

// Spawn launches a Claude Code session in the configured mode.
func Spawn(cfg *SpawnConfig) error {
	switch cfg.Mode {
	case store.ModeInteractive:
		return spawnInteractive(cfg)
	case store.ModeHeadless:
		return spawnHeadless(cfg)
	default:
		return fmt.Errorf("unknown mode: %s", cfg.Mode)
	}
}

func spawnInteractive(cfg *SpawnConfig) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", cfg.RunID, "bash", "-c", buildWrapperScript(cfg))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux new-session: %w", err)
	}

	// Show detach hint in the tmux status bar.
	_ = exec.Command("tmux", "set-option", "-t", cfg.RunID,
		"status-right", " Ctrl-B D to detach ").Run()

	// Auto-accept the workspace trust dialog (Enter on pre-selected "Yes").
	go func() {
		time.Sleep(3 * time.Second)
		_ = exec.Command("tmux", "send-keys", "-t", cfg.RunID, "Enter").Run()
	}()

	go monitorTmux(cfg)
	return nil
}

func monitorTmux(cfg *SpawnConfig) {
	for {
		time.Sleep(pollInterval)

		cmd := exec.Command("tmux", "has-session", "-t", cfg.RunID)
		if err := cmd.Run(); err != nil {
			exitCode := readExitCode(filepath.Join(filepath.Dir(cfg.RepoDir), ExitCodeFileName))
			if cfg.OnExit != nil {
				cfg.OnExit(exitCode)
			}
			return
		}
	}
}

func spawnHeadless(cfg *SpawnConfig) error {
	cmd := exec.Command("bash", "-c", buildWrapperScript(cfg))
	cmd.Dir = cfg.RepoDir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start headless: %w", err)
	}

	if cfg.Registry != nil {
		cfg.Registry.add(cfg.RunID, cmd.Process)
	}

	go func() {
		_ = cmd.Wait()

		if cfg.Registry != nil {
			cfg.Registry.remove(cfg.RunID)
		}

		exitCode := readExitCode(filepath.Join(filepath.Dir(cfg.RepoDir), ExitCodeFileName))
		if cfg.OnExit != nil {
			cfg.OnExit(exitCode)
		}
	}()

	return nil
}

// Stop terminates a running session by ID and mode.
func Stop(reg *Registry, runID string, mode store.Mode) {
	switch mode {
	case store.ModeInteractive:
		_ = exec.Command("tmux", "kill-session", "-t", runID).Run()
	case store.ModeHeadless:
		stopHeadless(reg, runID)
	}
}

func stopHeadless(reg *Registry, runID string) {
	if reg == nil {
		return
	}

	proc, ok := reg.get(runID)
	if !ok {
		return
	}

	_ = proc.Signal(syscall.SIGTERM)

	deadline := time.After(headlessStopWait)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			if reg.has(runID) {
				_ = proc.Signal(syscall.SIGKILL)
			}
			return
		case <-ticker.C:
			if !reg.has(runID) {
				return
			}
		}
	}
}

func readExitCode(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	s := strings.TrimSpace(string(data))
	code, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return code
}

var runIDSanitize = regexp.MustCompile(`[^a-zA-Z0-9-]`)

// GenerateRunID creates a unique run identifier from a name.
func GenerateRunID(name string) string {
	sanitized := runIDSanitize.ReplaceAllString(name, "-")
	if len(sanitized) > 30 {
		sanitized = sanitized[:30]
	}
	sanitized = strings.Trim(sanitized, "-")

	now := time.Now()
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		panic("failed to generate random suffix: " + err.Error())
	}

	ts := now.Format("20060102-150405")
	hexSuffix := hex.EncodeToString(suffix)

	if sanitized == "" {
		return "run-" + ts + "-" + hexSuffix
	}
	return sanitized + "-" + ts + "-" + hexSuffix
}
