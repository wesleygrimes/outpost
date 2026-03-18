package runner

import (
	"fmt"
	"os/exec"
	"strings"
)

// Extract unpacks a tarball into destDir, initializes a git repo to establish
// a base SHA for diffing, and optionally creates a branch.
func Extract(archivePath, destDir, branch string) (string, error) {
	tarCmd := exec.Command("tar", "xzf", archivePath, "-C", destDir)
	if out, err := tarCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("extracting archive: %w\n%s", err, out)
	}

	// Initialize a git repo so we can diff against the base state later.
	if out, err := exec.Command("git", "-C", destDir, "init").CombinedOutput(); err != nil {
		return "", fmt.Errorf("git init: %w\n%s", err, out)
	}

	if out, err := exec.Command("git", "-C", destDir, "add", "-A").CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add: %w\n%s", err, out)
	}

	commitCmd := exec.Command("git", "-C", destDir, "commit", "-m", "outpost: base snapshot")
	commitCmd.Env = append(commitCmd.Environ(),
		"GIT_AUTHOR_NAME=outpost",
		"GIT_AUTHOR_EMAIL=outpost@localhost",
		"GIT_COMMITTER_NAME=outpost",
		"GIT_COMMITTER_EMAIL=outpost@localhost",
	)

	if out, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit: %w\n%s", err, out)
	}

	if branch != "" {
		if out, err := exec.Command("git", "-C", destDir, "checkout", "-b", branch).CombinedOutput(); err != nil {
			return "", fmt.Errorf("git checkout -b %s: %w\n%s", branch, err, out)
		}
	}

	sha, err := headSHA(destDir)
	if err != nil {
		return "", err
	}

	return sha, nil
}

func headSHA(repoDir string) (string, error) {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
