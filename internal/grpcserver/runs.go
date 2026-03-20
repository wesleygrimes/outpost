package grpcserver

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesleygrimes/outpost/gen/outpost/v1"
	"github.com/wesleygrimes/outpost/internal/runner"
	"github.com/wesleygrimes/outpost/internal/store"
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

// Version is set by ldflags at build time.
var Version = "dev" //nolint:gochecknoglobals // injected via ldflags

// ServerDoctor returns server health diagnostics.
func (s *Server) ServerDoctor(_ context.Context, _ *outpostv1.ServerDoctorRequest) (*outpostv1.ServerDoctorResponse, error) {
	activeCount, _ := s.store.ActiveRuns()
	totalCount := len(s.store.List())

	return &outpostv1.ServerDoctorResponse{
		Version:         Version,
		Uptime:          formatUptime(time.Since(s.startTime)),
		DiskFree:        diskFree(s.runsDir),
		ClaudeInstalled: commandExists("claude"),
		TmuxInstalled:   commandExists("tmux"),
		ActiveRuns:      int32(activeCount),
		MaxRuns:         int32(s.cfg.MaxConcurrentRuns),
		TotalRuns:       int32(totalCount),
	}, nil
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

const diskFreeUnknown = "unknown"

func diskFree(path string) string {
	out, err := exec.Command("df", "-h", path).Output()
	if err != nil {
		return diskFreeUnknown
	}
	// Parse second line, fourth column (Available).
	lines := splitLines(string(out))
	if len(lines) < 2 {
		return diskFreeUnknown
	}
	fields := splitFields(lines[1])
	if len(fields) < 4 {
		return diskFreeUnknown
	}
	return fields[3] + " free"
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitFields(s string) []string {
	var fields []string
	inField := false
	start := 0
	for i := range len(s) {
		if s[i] == ' ' || s[i] == '\t' {
			if inField {
				fields = append(fields, s[start:i])
				inField = false
			}
		} else {
			if !inField {
				start = i
				inField = true
			}
		}
	}
	if inField {
		fields = append(fields, s[start:])
	}
	return fields
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
