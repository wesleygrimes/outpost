package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wesgrimes/outpost/internal/client"
)

// Handoff creates an archive of the working tree and submits it with a plan to the Outpost server.
func Handoff(args []string) {
	fs := flag.NewFlagSet("handoff", flag.ExitOnError)
	planPath := fs.String("plan", "", "path to the plan file (required)")
	mode := fs.String("mode", "interactive", "execution mode: interactive or headless")
	name := fs.String("name", "", "run name")
	branch := fs.String("branch", "", "git branch name")
	maxTurns := fs.Int("max-turns", 50, "max turns for headless mode")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: outpost handoff --plan FILE [--mode MODE] [--name NAME] [--branch BRANCH] [--max-turns N]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *planPath == "" {
		fmt.Fprintln(os.Stderr, "error: --plan is required")
		fs.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(*planPath); err != nil {
		fatalf("plan file: %v", err)
	}

	subdir := detectSubdir()

	archivePath, err := createArchive()
	if err != nil {
		fatalf("creating archive: %v", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	c, err := client.Load()
	if err != nil {
		fatalf("%v", err)
	}

	result, err := c.Handoff(&client.HandoffParams{
		PlanPath:    *planPath,
		ArchivePath: archivePath,
		Mode:        *mode,
		Name:        *name,
		Branch:      *branch,
		MaxTurns:    *maxTurns,
		Subdir:      subdir,
	})
	if err != nil {
		fatalf("%v", err)
	}

	fmt.Printf("id=%s\n", result.ID)
	fmt.Printf("status=%s\n", result.Status)

	if result.Attach != "" {
		fmt.Printf("attach=%s\n", result.Attach)
	}
}

// detectSubdir returns the git subdirectory prefix (for monorepo support).
// Returns empty string if at the repo root or outside a git repo.
func detectSubdir() string {
	out, err := exec.Command("git", "rev-parse", "--show-prefix").Output()
	if err != nil {
		return ""
	}

	return strings.TrimRight(strings.TrimSpace(string(out)), "/")
}

// createArchive builds a tar.gz of tracked and untracked files in the working directory.
// Returns the path to the temp archive file.
func createArchive() (string, error) {
	f, err := os.CreateTemp("", "outpost-archive-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	_ = f.Close()

	cmd := exec.Command("bash", "-c",
		"git ls-files -co --exclude-standard | tar czf "+f.Name()+" -T -")

	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("%w: %s", err, out)
	}

	return f.Name(), nil
}
