package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wesgrimes/outpost/internal/runner"
	"github.com/wesgrimes/outpost/internal/store"
)

const maxBundleSize = 2 << 30 // 2 GB

func (s *Server) handleHandoff(w http.ResponseWriter, r *http.Request) {
	if s.atCapacity(w) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBundleSize)

	if err := r.ParseMultipartForm(maxBundleSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	params, err := parseHandoffParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = params.bundleFile.Close() }()

	paths, err := prepareRunDir(params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	baseSHA, err := runner.Unbundle(paths.bundle, paths.repo, params.branch)
	if err != nil {
		cleanupRunDir(paths.dir)
		writeError(w, http.StatusInternalServerError, "unbundling: "+err.Error())

		return
	}

	// Start as pending; promoted to running after Spawn succeeds.
	run := &store.Run{
		ID:        params.runID,
		Name:      params.name,
		Mode:      params.mode,
		Status:    store.StatusPending,
		BaseSHA:   baseSHA,
		CreatedAt: time.Now(),
		Attach:    fmt.Sprintf("ssh outpost -t 'tmux attach -t %s'", params.runID),
		Branch:    params.branch,
		MaxTurns:  params.maxTurns,
		Dir:       paths.dir,
	}

	s.store.Add(run)

	err = runner.Spawn(&runner.SpawnConfig{
		RunID:    params.runID,
		RepoDir:  paths.repo,
		PlanPath: paths.plan,
		LogPath:  paths.log,
		Mode:     params.mode,
		MaxTurns: params.maxTurns,
		OnExit:   s.makeOnExit(params.runID, paths, baseSHA),
	})
	if err != nil {
		s.store.Update(params.runID, func(r *store.Run) {
			r.Status = store.StatusFailed
			now := time.Now()
			r.FinishedAt = &now
		})

		writeError(w, http.StatusInternalServerError, "spawning session: "+err.Error())

		return
	}

	// Spawn succeeded; promote to running.
	s.store.Update(params.runID, func(r *store.Run) {
		r.Status = store.StatusRunning
	})

	writeJSON(w, http.StatusAccepted, map[string]string{
		"id":     params.runID,
		"status": string(store.StatusRunning),
		"attach": run.Attach,
	})
}

func (s *Server) atCapacity(w http.ResponseWriter) bool {
	active := s.store.ActiveCount()
	if active < s.cfg.Server.MaxConcurrentRuns {
		return false
	}

	var running []map[string]string

	for _, run := range s.store.List() {
		if run.Status == store.StatusPending || run.Status == store.StatusRunning {
			running = append(running, map[string]string{
				"id":     run.ID,
				"name":   run.Name,
				"status": string(run.Status),
			})
		}
	}

	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":       "at capacity",
		"active":      active,
		"max":         s.cfg.Server.MaxConcurrentRuns,
		"active_runs": running,
	})

	return true
}

type handoffParams struct {
	runID      string
	name       string
	mode       string
	branch     string
	maxTurns   int
	plan       string
	bundleFile io.ReadCloser
}

type runPaths struct {
	dir    string
	repo   string
	bundle string
	plan   string
	log    string
	patch  string
}

func parseHandoffParams(r *http.Request) (*handoffParams, error) {
	plan := r.FormValue("plan")
	if plan == "" {
		return nil, errors.New("plan field is required")
	}

	bundleFile, _, err := r.FormFile("bundle")
	if err != nil {
		return nil, fmt.Errorf("bundle file is required: %w", err)
	}

	mode := r.FormValue("mode")
	if mode == "" {
		mode = runner.ModeInteractive
	}

	if err := runner.ValidateMode(mode); err != nil {
		_ = bundleFile.Close()
		return nil, err
	}

	maxTurns := 50
	if v := r.FormValue("max_turns"); v != "" {
		if parsed, parseErr := strconv.Atoi(v); parseErr == nil && parsed > 0 {
			maxTurns = parsed
		}
	}

	return &handoffParams{
		runID:      runner.GenerateRunID(r.FormValue("name")),
		name:       r.FormValue("name"),
		mode:       mode,
		branch:     r.FormValue("branch"),
		maxTurns:   maxTurns,
		plan:       plan,
		bundleFile: bundleFile,
	}, nil
}

func prepareRunDir(params *handoffParams) (*runPaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	runDir := filepath.Join(home, ".outpost", "runs", params.runID)

	paths := &runPaths{
		dir:    runDir,
		repo:   filepath.Join(runDir, "repo"),
		bundle: filepath.Join(runDir, "bundle.pack"),
		plan:   filepath.Join(runDir, "plan.md"),
		log:    filepath.Join(runDir, "output.log"),
		patch:  filepath.Join(runDir, "result.patch"),
	}

	if err := os.MkdirAll(paths.repo, 0o700); err != nil {
		return nil, fmt.Errorf("creating run directory: %w", err)
	}

	// Write bundle to disk.
	dst, err := os.Create(paths.bundle)
	if err != nil {
		cleanupRunDir(paths.dir)
		return nil, fmt.Errorf("creating bundle file: %w", err)
	}

	_, copyErr := io.Copy(dst, params.bundleFile)

	closeErr := dst.Close()

	if copyErr != nil {
		cleanupRunDir(paths.dir)
		return nil, fmt.Errorf("writing bundle: %w", copyErr)
	}

	if closeErr != nil {
		cleanupRunDir(paths.dir)
		return nil, fmt.Errorf("closing bundle file: %w", closeErr)
	}

	// Write plan with restrictive permissions.
	if err := os.WriteFile(paths.plan, []byte(params.plan), 0o600); err != nil {
		cleanupRunDir(paths.dir)
		return nil, fmt.Errorf("writing plan: %w", err)
	}

	return paths, nil
}

func cleanupRunDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("failed to clean up run directory %s: %v", dir, err)
	}
}

func (s *Server) makeOnExit(runID string, paths *runPaths, baseSHA string) func(int) {
	return func(exitCode int) {
		log.Printf("run %s exited with code %d", runID, exitCode)

		if patchErr := runner.GeneratePatch(paths.repo, baseSHA, paths.patch); patchErr != nil {
			log.Printf("run %s patch generation failed: %v", runID, patchErr)
		}

		finalSHA := headSHAFromDir(paths.repo)
		logTail := readTail(paths.log, 80)
		now := time.Now()

		s.store.Update(runID, func(r *store.Run) {
			if exitCode == 0 {
				r.Status = store.StatusComplete
			} else {
				r.Status = store.StatusFailed
			}

			r.FinalSHA = finalSHA
			r.FinishedAt = &now
			r.PatchReady = fileExists(paths.patch)
			r.LogTail = logTail
		})
	}
}

func headSHAFromDir(repoDir string) string {
	out, err := runGit(repoDir, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}

	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
