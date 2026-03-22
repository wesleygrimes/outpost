package grpcserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesleygrimes/outpost/gen/outpost/v1"
	"github.com/wesleygrimes/outpost/internal/runner"
	"github.com/wesleygrimes/outpost/internal/store"
)

// ConvertMode stops a running session and re-spawns it in the target mode.
// The Claude session is preserved via --continue so no context is lost.
func (s *Server) ConvertMode(_ context.Context, req *outpostv1.ConvertModeRequest) (*outpostv1.ConvertModeResponse, error) {
	targetMode := store.ModeFromProto(req.GetTargetMode())
	if targetMode != store.ModeInteractive && targetMode != store.ModeHeadless {
		return nil, status.Error(codes.InvalidArgument, "target_mode must be interactive or headless")
	}

	r, err := s.store.Get(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	if r.Status != store.StatusRunning && r.Status != store.StatusPending {
		return nil, status.Errorf(codes.FailedPrecondition, "run %s is not active (status=%s)", r.ID, r.Status)
	}

	if r.Mode == targetMode {
		return nil, status.Errorf(codes.InvalidArgument, "run %s is already in %s mode", r.ID, targetMode)
	}

	// Discover the forked session before stopping so we can resume it
	// deterministically instead of relying on --continue.
	repoDir := filepath.Join(r.Dir, "repo")
	forkedSessionID := r.ForkedSessionID
	if forkedSessionID == "" && r.SessionID != "" {
		if fid, err := runner.FindForkedSession(repoDir, r.SessionID); err == nil {
			forkedSessionID = fid
		}
	}

	// Mark as converting so the OnExit callback skips finalization.
	_ = s.store.Update(r.ID, func(run *store.Run) {
		run.Converting = true
	})

	// Graceful stop for interactive (gives Claude time to save state);
	// normal stop for headless.
	if r.Mode == store.ModeInteractive {
		runner.GracefulStopInteractive(r.ID)
	} else {
		runner.Stop(s.registry, r.ID, r.Mode)
	}

	// Brief wait for session state to flush to disk.
	time.Sleep(1 * time.Second)

	// Update the run's mode, forked session, and attach command.
	_ = s.store.Update(r.ID, func(run *store.Run) {
		run.Mode = targetMode
		run.Converting = false
		run.ForkedSessionID = forkedSessionID
		if targetMode == store.ModeInteractive {
			hostname, _ := os.Hostname()
			if s.cfg.SSHUser != "" {
				run.Attach = fmt.Sprintf("ssh -t %s sudo -u %s tmux attach-session -t %s", hostname, s.cfg.SSHUser, r.ID)
				run.AttachLocal = fmt.Sprintf("sudo -u %s tmux attach-session -t %s", s.cfg.SSHUser, r.ID)
			} else {
				run.Attach = fmt.Sprintf("ssh -t %s tmux attach-session -t %s", hostname, r.ID)
				run.AttachLocal = "tmux attach-session -t " + r.ID
			}
		} else {
			run.Attach = ""
			run.AttachLocal = ""
		}
	})

	// Re-spawn in the new mode, resuming the forked session if available.
	if err := s.respawnConverted(r, targetMode, forkedSessionID); err != nil {
		now := time.Now()
		_ = s.store.Update(r.ID, func(run *store.Run) {
			run.Status = store.StatusFailed
			run.FinishedAt = &now
		})
		return nil, status.Errorf(codes.Internal, "respawn: %v", err)
	}

	updated, _ := s.store.Get(r.ID)
	return &outpostv1.ConvertModeResponse{
		Run: store.RunToProto(updated),
	}, nil
}

// respawnConverted re-spawns a run in a new mode. It resumes the forked
// session directly when available, falling back to --continue.
func (s *Server) respawnConverted(r *store.Run, targetMode store.Mode, forkedSessionID string) error {
	runDir := r.Dir
	repoDir := filepath.Join(runDir, "repo")

	preTrustWorkspace(repoDir)

	cfg := &runner.SpawnConfig{
		RunID:    r.ID,
		RepoDir:  repoDir,
		LogPath:  filepath.Join(runDir, "output.log"),
		Mode:     targetMode,
		MaxTurns: r.MaxTurns,
		OnExit:   s.makeOnExit(r.ID, runDir, r.BaseSHA, r.SessionID),
		Registry: s.registry,
	}

	// Resume the forked session deterministically; fall back to --continue
	// if no forked session was discovered.
	if forkedSessionID != "" {
		cfg.SessionID = forkedSessionID
	} else {
		cfg.Continue = true
	}

	if err := runner.Spawn(cfg); err != nil {
		return err
	}

	_ = s.store.Update(r.ID, func(run *store.Run) {
		run.Status = store.StatusRunning
	})

	return nil
}
