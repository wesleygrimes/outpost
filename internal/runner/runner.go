// Package runner spawns and manages Claude Code sessions.
package runner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	RunID     string
	RepoDir   string
	PlanPath  string
	SessionID string // Claude session UUID to resume (mutually exclusive with PlanPath)
	Continue  bool   // Use --continue to resume the most recent session (for mode conversion)
	LogPath   string
	Mode      store.Mode
	MaxTurns  int
	OnExit    func(exitCode int)
	Registry  *Registry
}

// BuildClaudeCmd constructs the claude CLI invocation string.
// Priority: Continue > SessionID > PlanPath.
func BuildClaudeCmd(cfg *SpawnConfig) string {
	maxTurns := cfg.MaxTurns
	if maxTurns == 0 {
		maxTurns = DefaultMaxTurns
	}

	if cfg.Continue {
		return buildContinueCmd(cfg, maxTurns)
	}

	if cfg.SessionID != "" {
		return buildResumeCmd(cfg, maxTurns)
	}

	return buildPlanCmd(cfg, maxTurns)
}

func buildPlanCmd(cfg *SpawnConfig, maxTurns int) string {
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

func buildResumeCmd(cfg *SpawnConfig, maxTurns int) string {
	switch cfg.Mode {
	case store.ModeHeadless:
		// Two-phase: compact, then work
		compact := fmt.Sprintf(
			"claude --resume %s --fork-session --print -p %q --permission-mode bypassPermissions",
			cfg.SessionID,
			"/compact focus on the current task and next steps",
		)
		work := fmt.Sprintf(
			"claude --continue --print -p %q --permission-mode bypassPermissions --max-turns %d",
			"Continue working. Full conversation context preserved via session handoff.",
			maxTurns,
		)
		return compact + " && " + work

	case store.ModeInteractive, "":
		return fmt.Sprintf(
			"claude --resume %s --fork-session --permission-mode bypassPermissions --max-turns %d",
			cfg.SessionID, maxTurns,
		)
	}

	panic("unreachable: unknown mode " + string(cfg.Mode))
}

func buildContinueCmd(cfg *SpawnConfig, maxTurns int) string {
	switch cfg.Mode {
	case store.ModeHeadless:
		return fmt.Sprintf(
			"claude --continue --print -p %q --permission-mode bypassPermissions --max-turns %d",
			"Continue working. Mode converted to headless.",
			maxTurns,
		)
	case store.ModeInteractive, "":
		return fmt.Sprintf(
			"claude --continue --permission-mode bypassPermissions --max-turns %d",
			maxTurns,
		)
	}

	panic("unreachable: unknown mode " + string(cfg.Mode))
}

func buildWrapperScript(cfg *SpawnConfig) string {
	exitCodePath := filepath.Join(filepath.Dir(cfg.RepoDir), ExitCodeFileName)
	claudeCmd := BuildClaudeCmd(cfg)

	switch cfg.Mode {
	case store.ModeInteractive:
		// Interactive: run directly in the tmux PTY so the TUI renders
		// and the pre-seeded workspace trust is respected.
		return fmt.Sprintf(
			"cd %q && %s; echo $? > %q",
			cfg.RepoDir,
			claudeCmd,
			exitCodePath,
		)
	case store.ModeHeadless:
		// Headless: redirect stdout/stderr to log file.
		return fmt.Sprintf(
			"cd %q && %s > %q 2>&1; echo $? > %q",
			cfg.RepoDir, claudeCmd, cfg.LogPath,
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

// ComputePathHash converts an absolute directory path to the Claude projects
// path hash format: slashes and dots become dashes.
// e.g. /home/outpost/.outpost/runs/abc/repo -> -home-outpost-.outpost-runs-abc-repo
func ComputePathHash(dir string) string {
	h := strings.ReplaceAll(dir, string(filepath.Separator), "-")
	h = strings.ReplaceAll(h, ".", "-")
	return h
}

// FindForkedSession scans the Claude projects directory for a forked session
// JSONL file. It returns the session ID of the newest JSONL file that isn't
// the original session ID.
func FindForkedSession(repoDir, originalSessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}

	pathHash := ComputePathHash(repoDir)
	projectDir := filepath.Join(home, ".claude", "projects", pathHash)

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", fmt.Errorf("read project dir: %w", err)
	}

	var newestID string
	var newestTime time.Time

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(e.Name(), ".jsonl")
		if sessionID == originalSessionID {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if newestID == "" || info.ModTime().After(newestTime) {
			newestID = sessionID
			newestTime = info.ModTime()
		}
	}

	if newestID == "" {
		return "", fmt.Errorf("no forked session found in %s", projectDir)
	}

	return newestID, nil
}

// GenerateRunID creates a unique run identifier: YYYYMMDD-HHMMSS-XXXXXXXX.
func GenerateRunID() string {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		panic("failed to generate random suffix: " + err.Error())
	}

	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(suffix)
}
