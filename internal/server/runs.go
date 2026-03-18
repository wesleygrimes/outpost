package server

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wesgrimes/outpost/internal/store"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z@]|\x1b\][^\x07]*\x07|\x1b[()][AB012]`)

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	runs := s.store.List()
	if runs == nil {
		runs = []*store.Run{}
	}

	writeResponse(w, r, http.StatusOK, runs)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	run, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	// Refresh log tail for running sessions.
	if run.Status == store.StatusRunning {
		logPath := filepath.Join(run.Dir, "output.log")
		run.LogTail = readTail(logPath, 80)
	}

	writeResponse(w, r, http.StatusOK, run)
}

func (s *Server) handleGetPatch(w http.ResponseWriter, r *http.Request) {
	s.serveRunFile(w, r, "result.patch", "patch not ready")
}

func (s *Server) handleGetLog(w http.ResponseWriter, r *http.Request) {
	s.serveRunFile(w, r, "output.log", "log not available")
}

func (s *Server) serveRunFile(w http.ResponseWriter, r *http.Request, filename, notFoundMsg string) {
	id := r.PathValue("id")

	run, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	filePath := filepath.Join(run.Dir, filename)
	if !fileExists(filePath) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename="+id+"-"+filename)
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	run, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	// Kill the session if still running.
	if run.Status == store.StatusRunning || run.Status == store.StatusPending {
		capturePartialPatch(run)

		// Kill session (ignore errors if already dead).
		_ = exec.Command("tmux", "kill-session", "-t", id).Run()
	}

	now := time.Now()
	s.store.Update(id, func(r *store.Run) {
		r.Status = store.StatusKilled
		r.FinishedAt = &now
		r.PatchReady = fileExists(filepath.Join(r.Dir, "result.patch"))
		r.LogTail = readTail(filepath.Join(r.Dir, "output.log"), 80)
	})

	run, _ = s.store.Get(id)
	writeResponse(w, r, http.StatusOK, run)
}

func capturePartialPatch(run *store.Run) {
	repoDir := filepath.Join(run.Dir, "repo")
	patchPath := filepath.Join(run.Dir, "result.patch")

	if run.BaseSHA == "" {
		return
	}

	_ = exec.Command("git", "-C", repoDir, "add", "-A").Run()

	diffCmd := exec.Command("git", "-C", repoDir, "diff", "--cached", run.BaseSHA)

	out, err := diffCmd.Output()
	if err != nil || len(out) == 0 {
		return
	}

	_ = os.WriteFile(patchPath, out, 0o600)
}

func readTail(path string, lines int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	clean := stripANSI(string(data))
	allLines := strings.Split(clean, "\n")

	if len(allLines) <= lines {
		return clean
	}

	return strings.Join(allLines[len(allLines)-lines:], "\n")
}

// stripANSI removes ANSI escape sequences so log_tail is safe for JSON encoding.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func runGit(repoDir string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", repoDir}, args...)
	cmd := exec.Command("git", fullArgs...)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
