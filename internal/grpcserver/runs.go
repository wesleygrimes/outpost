package grpcserver

import (
	"context"
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
	"github.com/wesgrimes/outpost/internal/runner"
	"github.com/wesgrimes/outpost/internal/store"
)

// HealthCheck returns server health status.
func (s *Server) HealthCheck(_ context.Context, _ *outpostv1.HealthCheckRequest) (*outpostv1.HealthCheckResponse, error) {
	return &outpostv1.HealthCheckResponse{Status: "ok"}, nil
}

// GetRun returns a single run by ID, refreshing log_tail if running.
func (s *Server) GetRun(_ context.Context, req *outpostv1.GetRunRequest) (*outpostv1.GetRunResponse, error) {
	r, err := s.store.Get(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	if r.Status == store.StatusRunning {
		logPath := filepath.Join(s.runsDir, r.ID, "output.log")
		tail := readLogTail(logPath)
		r.LogTail = store.StripANSI(tail)
		_ = s.store.Update(r.ID, func(run *store.Run) {
			run.LogTail = r.LogTail
		})
	}

	return &outpostv1.GetRunResponse{Run: store.RunToProto(r)}, nil
}

// ListRuns returns all runs sorted by created_at descending.
func (s *Server) ListRuns(_ context.Context, _ *outpostv1.ListRunsRequest) (*outpostv1.ListRunsResponse, error) {
	runs := s.store.List()
	protoRuns := make([]*outpostv1.Run, 0, len(runs))
	for _, r := range runs {
		protoRuns = append(protoRuns, store.RunToProto(r))
	}
	return &outpostv1.ListRunsResponse{Runs: protoRuns}, nil
}

// DropRun stops a session and removes all data.
func (s *Server) DropRun(_ context.Context, req *outpostv1.DropRunRequest) (*outpostv1.DropRunResponse, error) {
	r, err := s.store.Get(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	runner.Stop(s.registry, r.ID, r.Mode)
	s.removeRunData(r)

	return &outpostv1.DropRunResponse{Id: r.ID}, nil
}

// CleanupRun removes a run's directory and store entry.
func (s *Server) CleanupRun(_ context.Context, req *outpostv1.CleanupRunRequest) (*outpostv1.CleanupRunResponse, error) {
	r, err := s.store.Get(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	s.removeRunData(r)

	return &outpostv1.CleanupRunResponse{Id: r.ID, Status: "cleaned"}, nil
}
