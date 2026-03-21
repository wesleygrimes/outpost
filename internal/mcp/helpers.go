package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wesleygrimes/outpost/internal/config"
	"github.com/wesleygrimes/outpost/internal/runner"
	"github.com/wesleygrimes/outpost/internal/store"
	"google.golang.org/grpc/status"
)

func boolPtr(b bool) *bool { return &b }

// runToMap converts a store.Run to a map for JSON serialization.
func runToMap(r *store.Run) map[string]any {
	m := map[string]any{
		"id":            r.ID,
		"name":          r.Name,
		"mode":          string(r.Mode),
		"status":        string(r.Status),
		"branch":        r.Branch,
		"base_sha":      r.BaseSHA,
		"final_sha":     r.FinalSHA,
		"created_at":    r.CreatedAt.Format(time.RFC3339),
		"patch_ready":   r.PatchReady,
		"session_ready": r.SessionReady,
		"max_turns":     r.MaxTurns,
	}
	if r.FinishedAt != nil {
		m["finished_at"] = r.FinishedAt.Format(time.RFC3339)
	}
	if r.Attach != "" {
		m["attach"] = r.Attach
	}
	if r.AttachLocal != "" {
		m["attach_local"] = r.AttachLocal
	}
	if r.LogTail != "" {
		m["log_tail"] = r.LogTail
	}
	if r.ForkedSessionID != "" {
		m["forked_session_id"] = r.ForkedSessionID
	}
	return m
}

// runSummary returns a compact summary of a run for dashboard views.
func runSummary(r *store.Run) map[string]any {
	m := map[string]any{
		"id":     r.ID,
		"status": string(r.Status),
		"mode":   string(r.Mode),
		"branch": r.Branch,
	}
	if r.Name != "" {
		m["name"] = r.Name
	}
	if r.PatchReady {
		m["patch_ready"] = true
	}
	m["created_at"] = r.CreatedAt.Format(time.RFC3339)
	if r.FinishedAt != nil {
		m["finished_at"] = r.FinishedAt.Format(time.RFC3339)
	}
	return m
}

// serverAddress returns the configured server address, or empty string.
func serverAddress() string {
	cfg, err := config.LoadClient()
	if err != nil {
		return ""
	}
	return cfg.Server
}

// humanizeGRPCError extracts the human-readable message from a gRPC error.
func humanizeGRPCError(err error) string {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if s, ok := status.FromError(e); ok && s.Code() != 0 {
			msg := s.Message()
			if i := strings.Index(msg, ": ["); i > 0 {
				return msg[:i]
			}
			return msg
		}
	}
	return err.Error()
}

// --- Session helpers (duplicated from cmd/handoff.go) ---

// readSessionJSONL reads and truncates the session JSONL for handoff.
func readSessionJSONL(sessionID string) ([]byte, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("user home: %w", err)
	}

	pathHash := runner.ComputePathHash(cwd)
	jsonlPath := filepath.Join(home, ".claude", "projects", pathHash, sessionID+".jsonl")

	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", jsonlPath, err)
	}

	return truncateBeforeHandoff(data), nil
}

// truncateBeforeHandoff removes the last human user message and everything
// after it from the JSONL.
func truncateBeforeHandoff(data []byte) []byte {
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))

	lastHumanIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if isHumanMessage(lines[i]) {
			lastHumanIdx = i
			break
		}
	}

	if lastHumanIdx <= 0 {
		return data
	}

	kept := bytes.Join(lines[:lastHumanIdx], []byte("\n"))
	kept = append(kept, '\n')
	return kept
}

// isHumanMessage returns true if the JSONL line is a user-typed message.
func isHumanMessage(line []byte) bool {
	var msg struct {
		Type    string `json:"type"`
		Message *struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return false
	}
	if msg.Type != "user" || msg.Message == nil {
		return false
	}
	trimmed := bytes.TrimSpace(msg.Message.Content)
	return len(trimmed) > 0 && trimmed[0] == '"'
}

// createArchive creates a tar.gz of git-tracked files and returns the temp path.
func createArchive() (string, error) {
	f, err := os.CreateTemp("", "outpost-archive-*.tar.gz")
	if err != nil {
		return "", err
	}
	archivePath := f.Name()
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("git ls-files -co --exclude-standard | tar czf %q -T -", archivePath))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(archivePath)
		return "", fmt.Errorf("tar: %w", err)
	}

	return archivePath, nil
}

// downloadSession downloads the forked session JSONL to the local Claude
// projects directory.
func downloadSession(runID, forkedSessionID string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}

	pathHash := runner.ComputePathHash(cwd)
	destDir := filepath.Join(home, ".claude", "projects", pathHash)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create projects dir: %w", err)
	}

	destPath := filepath.Join(destDir, forkedSessionID+".jsonl")
	return destPath, nil
}
