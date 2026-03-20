package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wesgrimes/outpost/internal/grpcclient"
	"github.com/wesgrimes/outpost/internal/runner"
)

// Pickup downloads a completed patch and cleans up the remote run.
func Pickup(args []string) error {
	jsonOut, args := hasFlag(args, "--json")

	if len(args) < 1 {
		return errors.New("usage: outpost pickup <run_id>")
	}
	id := args[0]

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	ctx := context.Background()

	r, err := client.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if !r.PatchReady {
		return fmt.Errorf("no patch available for run %s (status=%s)", id, r.Status)
	}

	patchDir := ".outpost/patches"
	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		return fmt.Errorf("create patches dir: %w", err)
	}

	patchPath := filepath.Join(patchDir, id+".patch")

	fmt.Fprintln(os.Stderr, "downloading patch...")
	if err := client.DownloadPatch(ctx, id, patchPath); err != nil {
		return fmt.Errorf("download patch: %w", err)
	}

	var sessionID string
	if r.SessionReady && r.ForkedSessionID != "" {
		if err := downloadSession(ctx, client, id, r.ForkedSessionID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: session download failed: %v\n", err)
		} else {
			sessionID = r.ForkedSessionID
		}
	}

	if err := client.CleanupRun(ctx, id); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cleanup failed: %v\n", err)
	}

	if jsonOut {
		result := map[string]string{
			"id":    id,
			"patch": patchPath,
		}
		if sessionID != "" {
			result["session"] = sessionID
		}
		return printJSON(result)
	}

	printHeader()
	fmt.Println()
	printField("Patch:", patchPath)
	fmt.Println()

	diffStat := exec.Command("git", "diff", "--stat", patchPath)
	diffStat.Stdout = os.Stdout
	diffStat.Stderr = os.Stderr
	_ = diffStat.Run()

	if sessionID != "" {
		fmt.Println()
		printField("Session:", sessionID)
	}

	return nil
}

// downloadSession downloads the forked session JSONL to the local Claude
// projects directory so the user can `claude --resume <id>`.
func downloadSession(ctx context.Context, client *grpcclient.Client, runID, forkedSessionID string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("user home: %w", err)
	}

	pathHash := runner.ComputePathHash(cwd)
	destPath := filepath.Join(home, ".claude", "projects", pathHash, forkedSessionID+".jsonl")

	return client.DownloadSession(ctx, runID, destPath)
}
