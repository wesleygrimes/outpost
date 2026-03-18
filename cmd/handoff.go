package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/wesgrimes/outpost/internal/grpcclient"
	"github.com/wesgrimes/outpost/internal/runner"
	"github.com/wesgrimes/outpost/internal/store"
)

// Handoff creates an archive and streams it to the Outpost server.
func Handoff(args []string) error {
	fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
	planPath := fs.String("plan", "", "path to plan file")
	mode := fs.String("mode", "interactive", "run mode (interactive or headless)")
	name := fs.String("name", "", "run name")
	branch := fs.String("branch", "", "git branch")
	maxTurns := fs.Int("max-turns", runner.DefaultMaxTurns, "max turns")
	subdir := fs.String("subdir", "", "subdirectory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *planPath == "" {
		return errors.New("--plan is required")
	}

	plan, err := os.ReadFile(*planPath)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
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

	result, err := client.Handoff(context.Background(), archivePath, &grpcclient.HandoffMeta{
		Plan:     string(plan),
		Mode:     store.ModeToProto(store.Mode(*mode)),
		Name:     *name,
		Branch:   *branch,
		MaxTurns: int32(*maxTurns),
		Subdir:   *subdir,
	}, func(sent, total int64) {
		fmt.Fprintf(os.Stderr, "\ruploading... %s / %s",
			formatMB(sent), formatMB(total))
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr)

	fmt.Printf("id=%s\n", result.ID)
	fmt.Printf("status=%s\n", result.Status)
	if result.Attach != "" {
		fmt.Printf("attach=%s\n", result.Attach)
	}

	return nil
}

func formatMB(b int64) string {
	return strconv.FormatFloat(float64(b)/1024/1024, 'f', 1, 64) + " MB"
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
