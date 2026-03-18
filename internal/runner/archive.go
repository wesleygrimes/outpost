package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Extract unpacks a tar.gz archive into destDir, initializes a git repo, and returns the base SHA.
func Extract(archivePath, destDir, branch string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	cmd := exec.Command("tar", "xzf", archivePath, "-C", destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("extract: %w: %s", err, out)
	}

	if err := runGitInDir(destDir, "init"); err != nil {
		return "", fmt.Errorf("git init: %w", err)
	}

	if err := runGitInDir(destDir, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}

	if err := runGitInDir(destDir, "commit", "-m", "outpost: initial commit",
		"--author", "outpost <outpost@local>"); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	if branch != "" {
		if err := runGitInDir(destDir, "checkout", "-b", branch); err != nil {
			return "", fmt.Errorf("git checkout -b: %w", err)
		}
	}

	sha, err := gitOutput(destDir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("rev-parse: %w", err)
	}

	return strings.TrimSpace(sha), nil
}

// GitHeadSHA returns the HEAD commit SHA for a repo directory.
func GitHeadSHA(dir string) (string, error) {
	out, err := gitOutput(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func runGitInDir(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=outpost",
		"GIT_AUTHOR_EMAIL=outpost@local",
		"GIT_COMMITTER_NAME=outpost",
		"GIT_COMMITTER_EMAIL=outpost@local",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
