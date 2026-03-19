package grpcserver

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
	"github.com/wesgrimes/outpost/internal/runner"
	"github.com/wesgrimes/outpost/internal/store"
)

// Handoff receives a plan and archive via client streaming, then spawns a session.
func (s *Server) Handoff(stream outpostv1.OutpostService_HandoffServer) error {
	meta, err := s.recvHandoffMeta(stream)
	if err != nil {
		return err
	}

	mode := store.ModeFromProto(meta.GetMode())
	if mode != store.ModeInteractive && mode != store.ModeHeadless {
		return status.Error(codes.InvalidArgument, "mode must be interactive or headless")
	}

	if err := s.checkCapacity(); err != nil {
		return err
	}

	runID := runner.GenerateRunID(meta.GetName())
	runDir := filepath.Join(s.runsDir, runID)

	if err := os.MkdirAll(filepath.Join(runDir, "repo"), 0o755); err != nil {
		return status.Errorf(codes.Internal, "create run dir: %v", err)
	}

	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			_ = os.RemoveAll(runDir)
		}
	}()

	if err := os.WriteFile(filepath.Join(runDir, "plan.md"), []byte(meta.GetPlan()), 0o644); err != nil {
		return status.Errorf(codes.Internal, "write plan: %v", err)
	}

	if err := s.recvArchiveChunks(stream, filepath.Join(runDir, "archive.tar.gz")); err != nil {
		return err
	}

	baseSHA, err := runner.Extract(filepath.Join(runDir, "archive.tar.gz"), filepath.Join(runDir, "repo"), meta.GetBranch())
	if err != nil {
		return status.Errorf(codes.Internal, "extract archive: %v", err)
	}

	run := s.buildRun(runID, meta, mode, baseSHA, runDir)
	s.store.Add(run)

	if err := s.spawnRun(run, runDir, baseSHA); err != nil {
		return err
	}

	cleanupNeeded = false

	return stream.SendAndClose(&outpostv1.HandoffResponse{
		Id:     runID,
		Status: outpostv1.RunStatus_RUN_STATUS_RUNNING,
		Attach: run.Attach,
	})
}

func (s *Server) recvHandoffMeta(stream outpostv1.OutpostService_HandoffServer) (*outpostv1.HandoffMetadata, error) {
	first, err := stream.Recv()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "recv metadata: %v", err)
	}

	meta := first.GetMetadata()
	if meta == nil {
		return nil, status.Error(codes.InvalidArgument, "first message must be metadata")
	}

	if meta.GetPlan() == "" {
		return nil, status.Error(codes.InvalidArgument, "plan is required")
	}

	return meta, nil
}

func (s *Server) checkCapacity() error {
	count, active := s.store.ActiveRuns()
	if count < s.cfg.MaxConcurrentRuns {
		return nil
	}

	var activeList []map[string]string
	for i := range active {
		activeList = append(activeList, map[string]string{
			"id":     active[i].ID,
			"status": string(active[i].Status),
		})
	}
	detail, err := json.Marshal(activeList)
	if err != nil {
		return status.Errorf(codes.ResourceExhausted, "at capacity (%d/%d active runs)", count, s.cfg.MaxConcurrentRuns)
	}
	return status.Errorf(codes.ResourceExhausted, "at capacity (%d/%d active runs): %s",
		count, s.cfg.MaxConcurrentRuns, detail)
}

func (s *Server) recvArchiveChunks(stream outpostv1.OutpostService_HandoffServer, archivePath string) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return status.Errorf(codes.Internal, "create archive: %v", err)
	}
	defer func() { _ = f.Close() }()

	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "recv chunk: %v", err)
		}

		data := msg.GetData()
		if data == nil {
			return status.Error(codes.InvalidArgument, "expected data chunk after metadata")
		}

		if _, err := f.Write(data); err != nil {
			return status.Errorf(codes.Internal, "write chunk: %v", err)
		}
	}
}

func (s *Server) buildRun(runID string, meta *outpostv1.HandoffMetadata, mode store.Mode, baseSHA, runDir string) *store.Run {
	run := &store.Run{
		ID:        runID,
		Name:      meta.GetName(),
		Mode:      mode,
		Status:    store.StatusPending,
		BaseSHA:   baseSHA,
		CreatedAt: time.Now(),
		Branch:    meta.GetBranch(),
		MaxTurns:  int(meta.GetMaxTurns()),
		Subdir:    meta.GetSubdir(),
		Dir:       runDir,
	}

	if mode == store.ModeInteractive {
		hostname, _ := os.Hostname()
		if s.cfg.SSHUser != "" {
			run.Attach = fmt.Sprintf("ssh -t %s sudo -u %s tmux attach-session -t %s", hostname, s.cfg.SSHUser, runID)
		} else {
			run.Attach = fmt.Sprintf("ssh -t %s tmux attach-session -t %s", hostname, runID)
		}
	}

	return run
}

// preTrustWorkspace writes the hasTrustDialogAccepted flag to
// ~/.claude/settings.json so the interactive trust prompt is skipped.
func preTrustWorkspace(repoDir string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")

	var settings map[string]any
	if data, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	projects, _ := settings["projects"].(map[string]any)
	if projects == nil {
		projects = make(map[string]any)
	}

	proj, _ := projects[repoDir].(map[string]any)
	if proj == nil {
		proj = make(map[string]any)
	}
	proj["hasTrustDialogAccepted"] = true
	projects[repoDir] = proj
	settings["projects"] = projects

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(settingsPath), 0o755)
	_ = os.WriteFile(settingsPath, data, 0o600)
}

func (s *Server) spawnRun(run *store.Run, runDir, baseSHA string) error {
	preTrustWorkspace(s.runsDir)

	cfg := &runner.SpawnConfig{
		RunID:    run.ID,
		RepoDir:  filepath.Join(runDir, "repo"),
		PlanPath: filepath.Join(runDir, "plan.md"),
		LogPath:  filepath.Join(runDir, "output.log"),
		Mode:     run.Mode,
		MaxTurns: run.MaxTurns,
		OnExit:   s.makeOnExit(run.ID, runDir, baseSHA),
		Registry: s.registry,
	}

	if err := runner.Spawn(cfg); err != nil {
		now := time.Now()
		_ = s.store.Update(run.ID, func(r *store.Run) {
			r.Status = store.StatusFailed
			r.FinishedAt = &now
		})
		return status.Errorf(codes.Internal, "spawn: %v", err)
	}

	_ = s.store.Update(run.ID, func(r *store.Run) {
		r.Status = store.StatusRunning
	})

	return nil
}

func (s *Server) makeOnExit(runID, runDir, baseSHA string) func(int) {
	repoDir := filepath.Join(runDir, "repo")
	logPath := filepath.Join(runDir, "output.log")
	patchPath := filepath.Join(runDir, "result.patch")

	return func(exitCode int) {
		_ = runner.GeneratePatch(repoDir, baseSHA, patchPath)

		finalSHA, _ := runner.GitHeadSHA(repoDir)
		logTail := readLogTail(logPath)

		st := store.StatusComplete
		if exitCode != 0 {
			st = store.StatusFailed
		}

		patchReady := false
		if info, err := os.Stat(patchPath); err == nil && info.Size() > 0 {
			patchReady = true
		}

		now := time.Now()
		_ = s.store.Update(runID, func(r *store.Run) {
			r.Status = st
			r.FinalSHA = finalSHA
			r.FinishedAt = &now
			r.LogTail = store.StripANSI(logTail)
			r.PatchReady = patchReady
		})
	}
}

const logTailLines = 80

func readLogTail(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > logTailLines {
		lines = lines[len(lines)-logTailLines:]
	}

	return strings.Join(lines, "\n")
}
