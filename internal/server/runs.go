package server

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wesgrimes/outpost/internal/store"
)

func (s *Server) handleListRuns(w http.ResponseWriter, _ *http.Request) {
	runs := s.store.List()
	if runs == nil {
		runs = []*store.Run{}
	}

	writeJSON(w, http.StatusOK, runs)
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

	writeJSON(w, http.StatusOK, run)
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

	// Kill the zellij session if still running.
	if run.Status == store.StatusRunning || run.Status == store.StatusPending {
		capturePartialPatch(run)

		// Kill session (ignore errors if already dead).
		_ = exec.Command("zellij", "kill-session", id).Run()
	}

	now := time.Now()
	s.store.Update(id, func(r *store.Run) {
		r.Status = store.StatusKilled
		r.FinishedAt = &now
		r.PatchReady = fileExists(filepath.Join(r.Dir, "result.patch"))
		r.LogTail = readTail(filepath.Join(r.Dir, "output.log"), 80)
	})

	run, _ = s.store.Get(id)
	writeJSON(w, http.StatusOK, run)
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

	allLines := strings.Split(string(data), "\n")
	if len(allLines) <= lines {
		return string(data)
	}

	return strings.Join(allLines[len(allLines)-lines:], "\n")
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
