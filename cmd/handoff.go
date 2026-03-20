package cmd

import (
	"context"
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
		return err
	}

	pb.Done(fmt.Sprintf("Streamed (%s in %s)", ui.FormatBytes(lastTotal), formatDuration(pb.Elapsed())))

	if *jsonOut {
		return printJSON(map[string]string{
			"id":     result.ID,
			"status": string(result.Status),
			"attach": result.Attach,
		})
	}

	ui.Header("Handoff")
	ui.Errln()
	ui.Field("Run", ui.Amber(result.ID))
	ui.Field("Status", string(result.Status))
	if result.Attach != "" {
		ui.Field("Attach", result.Attach)
	}
	ui.Errln()
	ui.Hint("Watch", "outpost status "+result.ID+" --follow")
	ui.Hint("Logs", "outpost logs "+result.ID+" --tail")

	return nil
}

// readSessionJSONL finds and reads the session JSONL file from the Claude
// projects directory for the current working directory.
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

	return data, nil
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
