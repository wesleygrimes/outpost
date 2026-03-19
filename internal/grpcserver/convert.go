package grpcserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
	"github.com/wesgrimes/outpost/internal/runner"
	"github.com/wesgrimes/outpost/internal/store"
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

	// Mark as converting so the OnExit callback skips finalization.
	_ = s.store.Update(r.ID, func(run *store.Run) {
		run.Converting = true
	})

	// Stop the current process.
	runner.Stop(s.registry, r.ID, r.Mode)

	// Brief wait for the process to fully exit and Claude to save session state.
	time.Sleep(2 * time.Second)

	// Update the run's mode and attach command.
	_ = s.store.Update(r.ID, func(run *store.Run) {
		run.Mode = targetMode
		run.Converting = false
		if targetMode == store.ModeInteractive {
			hostname, _ := os.Hostname()
			if s.cfg.SSHUser != "" {
				run.Attach = fmt.Sprintf("ssh -t %s sudo -u %s tmux attach-session -t %s", hostname, s.cfg.SSHUser, r.ID)
			} else {
				run.Attach = fmt.Sprintf("ssh -t %s tmux attach-session -t %s", hostname, r.ID)
			}
		} else {
			run.Attach = ""
		}
	})

	// Re-spawn in the new mode with --continue to resume the Claude session.
	if err := s.respawnConverted(r, targetMode); err != nil {
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

// respawnConverted re-spawns a run in a new mode using --continue.
func (s *Server) respawnConverted(r *store.Run, targetMode store.Mode) error {
	runDir := r.Dir
	repoDir := filepath.Join(runDir, "repo")

	preTrustWorkspace(repoDir)

	cfg := &runner.SpawnConfig{
		RunID:   r.ID,
		RepoDir: repoDir,
		LogPath: filepath.Join(runDir, "output.log"),
		Mode:    targetMode,
		// Use Continue flag to resume the most recent session in this project dir.
		Continue: true,
		MaxTurns: r.MaxTurns,
		OnExit:   s.makeOnExit(r.ID, runDir, r.BaseSHA, r.SessionID),
		Registry: s.registry,
	}

	if err := runner.Spawn(cfg); err != nil {
		return err
	}

	_ = s.store.Update(r.ID, func(run *store.Run) {
		run.Status = store.StatusRunning
	})

	return nil
}
