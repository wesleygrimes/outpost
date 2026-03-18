package grpcserver

import (
	"io"
	"os"
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
)

const chunkSize = 64 * 1024 // 64 KiB

// DownloadPatch streams a patch file in 64 KiB chunks.
func (s *Server) DownloadPatch(req *outpostv1.DownloadPatchRequest, stream outpostv1.OutpostService_DownloadPatchServer) error {
	if _, err := s.store.Get(req.GetId()); err != nil {
		return status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	patchPath := filepath.Join(s.runsDir, req.GetId(), "result.patch")
	f, err := os.Open(patchPath)
	if err != nil {
		return status.Errorf(codes.NotFound, "patch not found for run %s", req.GetId())
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, chunkSize)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&outpostv1.DownloadPatchResponse{Data: buf[:n]}); sendErr != nil {
				return sendErr
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return status.Errorf(codes.Internal, "read patch: %v", readErr)
		}
	}
}
