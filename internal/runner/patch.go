package runner

import (
	"fmt"
	"os"
	"os/exec"
)

// GeneratePatch creates a diff from baseSHA to working tree and writes it to patchPath.
func GeneratePatch(repoDir, baseSHA, patchPath string) error {
	if err := runGitInDir(repoDir, "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	cmd := exec.Command("git", "diff", "--cached", baseSHA)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	if err := os.WriteFile(patchPath, out, 0o600); err != nil {
		return fmt.Errorf("write patch: %w", err)
	}

	return nil
}
