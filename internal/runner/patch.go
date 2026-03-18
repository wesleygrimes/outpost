package runner

import (
	"fmt"
	"os"
	"os/exec"
)

// GeneratePatch stages all changes in repoDir and writes a diff against baseSHA to patchPath.
func GeneratePatch(repoDir, baseSHA, patchPath string) error {
	addCmd := exec.Command("git", "-C", repoDir, "add", "-A")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add -A: %w\n%s", err, out)
	}

	diffCmd := exec.Command("git", "-C", repoDir, "diff", "--cached", baseSHA)
	out, err := diffCmd.Output()
	if err != nil {
		return fmt.Errorf("git diff --cached: %w", err)
	}

	if err := os.WriteFile(patchPath, out, 0o600); err != nil {
		return fmt.Errorf("writing patch: %w", err)
	}

	return nil
}
