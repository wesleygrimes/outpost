package grpcserver

import (
	"io"
	"os"
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesleygrimes/outpost/gen/outpost/v1"
	"github.com/wesleygrimes/outpost/internal/runner"
)

// DownloadSession streams a forked session JSONL file in 64 KiB chunks.
func (s *Server) DownloadSession(req *outpostv1.DownloadSessionRequest, stream outpostv1.OutpostService_DownloadSessionServer) error {
	r, err := s.store.Get(req.GetId())
	if err != nil {
		return status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	if r.ForkedSessionID == "" {
		return status.Errorf(codes.FailedPrecondition, "no forked session available for run %s", req.GetId())
	}

	repoDir := filepath.Join(s.runsDir, req.GetId(), "repo")
	pathHash := runner.ComputePathHash(repoDir)

	home, err := os.UserHomeDir()
	if err != nil {
		return status.Errorf(codes.Internal, "user home: %v", err)
	}

	jsonlPath := filepath.Join(home, ".claude", "projects", pathHash, r.ForkedSessionID+".jsonl")
	f, err := os.Open(jsonlPath)
	if err != nil {
		return status.Errorf(codes.NotFound, "session file not found for run %s", req.GetId())
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, chunkSize)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&outpostv1.DownloadSessionResponse{Data: buf[:n]}); sendErr != nil {
				return sendErr
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return status.Errorf(codes.Internal, "read session: %v", readErr)
		}
	}
}
