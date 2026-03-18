package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Pickup downloads a completed patch and cleans up the remote run.
func Pickup(args []string) error {
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

	fmt.Printf("patch=%s\n", patchPath)
	fmt.Println()

	diffStat := exec.Command("git", "diff", "--stat", patchPath)
	diffStat.Stdout = os.Stdout
	diffStat.Stderr = os.Stderr
	_ = diffStat.Run()

	if err := client.CleanupRun(ctx, id); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cleanup failed: %v\n", err)
	}

	return nil
}
