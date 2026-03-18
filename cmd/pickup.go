package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wesgrimes/outpost/internal/client"
)

// Pickup downloads a completed run's patch, creates a git worktree, and applies the changes.
func Pickup(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: outpost pickup <run_id>")
		fmt.Fprintln(os.Stderr, "Run 'outpost status' to find run IDs.")
		os.Exit(1)
	}

	runID := args[0]

	c, err := client.Load()
	if err != nil {
		fatalf("%v", err)
	}

	run, err := c.GetRun(runID)
	if err != nil {
		fatalf("%v", err)
	}

	if !run.PatchReady {
		fatalf("run %s is %s, patch not ready", runID, run.Status)
	}

	patchPath := downloadPatch(c, runID)
	defer func() { _ = os.Remove(patchPath) }()

	worktreeAbs, branchName := applyToWorktree(runID, patchPath, run.Subdir)

	// Clean up remote run (best-effort).
	if cleanupErr := c.Cleanup(runID); cleanupErr != nil {
		fmt.Fprintf(os.Stderr, "warning: remote cleanup failed: %v\n", cleanupErr)
	}

	fmt.Printf("worktree=%s\n", worktreeAbs)
	fmt.Printf("branch=%s\n", branchName)

	if stat, statErr := exec.Command("git", "-C", worktreeAbs, "diff", "--stat", "HEAD~1").Output(); statErr == nil {
		fmt.Printf("\n%s", string(stat))
	}
}

func downloadPatch(c *client.Client, runID string) string {
	patchFile, err := os.CreateTemp("", "outpost-patch-*.patch")
	if err != nil {
		fatalf("creating temp file: %v", err)
	}

	patchPath := patchFile.Name()
	_ = patchFile.Close()

	if err := c.DownloadPatch(runID, patchPath); err != nil {
		_ = os.Remove(patchPath)
		fatalf("downloading patch: %v", err)
	}

	info, err := os.Stat(patchPath)
	if err != nil || info.Size() == 0 {
		_ = os.Remove(patchPath)
		fatalf("patch is empty, nothing to apply")
	}

	return patchPath
}

func applyToWorktree(runID, patchPath, subdir string) (worktreeAbs, branchName string) {
	worktreeRel := filepath.Join("..", "outpost-"+runID)
	branchName = "outpost/" + runID

	out, err := exec.Command("git", "worktree", "add", "-b", branchName, worktreeRel).CombinedOutput()
	if err != nil {
		fatalf("creating worktree: %v\n%s", err, out)
	}

	worktreeAbs, err = filepath.Abs(worktreeRel)
	if err != nil {
		fatalf("resolving worktree path: %v", err)
	}

	// Apply patch.
	applyArgs := []string{"-C", worktreeAbs, "apply"}
	if subdir != "" {
		applyArgs = append(applyArgs, "--directory="+subdir)
	}

	applyArgs = append(applyArgs, patchPath)

	if out, err = exec.Command("git", applyArgs...).CombinedOutput(); err != nil {
		_ = exec.Command("git", "worktree", "remove", worktreeAbs).Run()
		fatalf("applying patch: %v\n%s", err, out)
	}

	// Stage and commit.
	if out, err = exec.Command("git", "-C", worktreeAbs, "add", "-A").CombinedOutput(); err != nil {
		_ = exec.Command("git", "worktree", "remove", worktreeAbs).Run()
		fatalf("staging changes: %v\n%s", err, out)
	}

	commitMsg := "outpost: apply run " + runID
	if out, err = exec.Command("git", "-C", worktreeAbs, "commit", "-m", commitMsg).CombinedOutput(); err != nil {
		_ = exec.Command("git", "worktree", "remove", worktreeAbs).Run()
		fatalf("committing: %v\n%s", err, out)
	}

	return worktreeAbs, branchName
}
