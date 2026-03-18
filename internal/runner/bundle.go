package runner

import (
	"fmt"
	"os/exec"
	"strings"
)

// Unbundle clones a git bundle into destDir and optionally checks out branch.
// Returns the base SHA (HEAD after unbundle).
func Unbundle(bundlePath, destDir, branch string) (string, error) {
	cloneCmd := exec.Command("git", "clone", bundlePath, destDir)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone bundle: %w\n%s", err, out)
	}

	if branch != "" {
		checkoutCmd := exec.Command("git", "-C", destDir, "checkout", "-b", branch)
		if out, err := checkoutCmd.CombinedOutput(); err != nil {
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
