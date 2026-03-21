package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/runner"
	"github.com/wesleygrimes/outpost/internal/store"
	"github.com/wesleygrimes/outpost/internal/ui"
)

// Handoff creates an archive and streams it to the Outpost server.
func Handoff(args []string) error {
	fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
	sessionID := fs.String("session-id", "", "Claude session UUID to resume (required)")
	mode := fs.String("mode", "interactive", "run mode (interactive or headless)")
	name := fs.String("name", "", "run name")
	branch := fs.String("branch", "", "git branch")
	maxTurns := fs.Int("max-turns", runner.DefaultMaxTurns, "max turns")
	subdir := fs.String("subdir", "", "subdirectory")
	jsonOut := fs.Bool("json", false, "output JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		return errors.New("--session-id is required")
	}

	sessionJSONL, err := readSessionJSONL(*sessionID)
	if err != nil {
		return fmt.Errorf("read session: %w", err)
	}

	archivePath, err := createArchive()
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	// Preflight: check server has capacity before streaming the archive.
	ctx := context.Background()
	doc, err := client.ServerDoctor(ctx)
	if err != nil {
		return fmt.Errorf("preflight check: %w", err)
	}
	if doc.ActiveRuns >= doc.MaxRuns {
		ui.Errln()
		ui.Errln("  " + ui.Fail(fmt.Sprintf("Handoff failed: server at capacity (%d/%d active runs).", doc.ActiveRuns, doc.MaxRuns)))
		ui.Errln()
		ui.Fix("outpost status")
		ui.Fix("outpost drop <run_id>")
		return &DisplayedError{Err: fmt.Errorf("handoff failed: server at capacity (%d/%d active runs)", doc.ActiveRuns, doc.MaxRuns)}
	}

	ui.Header("Handoff")
	ui.Errln()

	pb := ui.NewProgress("Streaming to server")
	var lastTotal int64
	result, err := client.Handoff(context.Background(), archivePath, &grpcclient.HandoffMeta{
		SessionID:    *sessionID,
		SessionJSONL: sessionJSONL,
		Mode:         store.ModeToProto(store.Mode(*mode)),
		Name:         *name,
		Branch:       *branch,
		MaxTurns:     int32(*maxTurns),
		Subdir:       *subdir,
	}, func(sent, total int64) {
		lastTotal = total
		pb.Update(sent, total)
	})
	if err != nil {
		pb.Done(fmt.Sprintf("Streamed (%s)", ui.FormatBytes(lastTotal)))
		ui.Errln()
		ui.Errln("  " + ui.Fail("Handoff failed: "+humanizeGRPCError(err)))
		return &DisplayedError{Err: fmt.Errorf("handoff failed: %s", humanizeGRPCError(err))}
	}

	pb.Done(fmt.Sprintf("Streamed (%s in %s)", ui.FormatBytes(lastTotal), formatDuration(pb.Elapsed())))

	attach := attachCmd(result.Attach, result.AttachLocal)

	if *jsonOut {
		return printJSON(map[string]string{
			"id":     result.ID,
			"status": string(result.Status),
			"attach": attach,
		})
	}

	status := store.StatusFromProto(result.Status)
	ui.Errln()
	ui.Field("Run", ui.Amber(result.ID))
	ui.Field("Status", ui.Check(string(status)))
	if attach != "" {
		ui.Field("Attach", attach)
	}
	ui.Errln()
	ui.Hint("Watch", "outpost status "+result.ID+" --follow")
	ui.Hint("Logs", "outpost logs "+result.ID+" --tail")

	return nil
}

// readSessionJSONL finds and reads the session JSONL file from the Claude
// projects directory for the current working directory. The returned bytes
// are truncated to exclude the user turn that triggered the handoff so the
// remote session resumes cleanly without re-executing the handoff.
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
// after it from the JSONL. This prevents the remote session from seeing the
// handoff request and recursively trying to hand off again.
//
// Human messages have {"type":"user","message":{"content":"<string>"}}.
// Tool results also have type "user" but their content is an array, not a string.
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

// isHumanMessage returns true if the JSONL line is a user-typed message
// (as opposed to a tool result or other system message).
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
	// Human messages have a string content; tool results have an array.
	trimmed := bytes.TrimSpace(msg.Message.Content)
	return len(trimmed) > 0 && trimmed[0] == '"'
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

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
