package grpcserver

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesleygrimes/outpost/gen/outpost/v1"
	"github.com/wesleygrimes/outpost/internal/store"
)

const logPollInterval = 500 * time.Millisecond

// TailLogs streams log lines for a run, optionally following until completion.
func (s *Server) TailLogs(req *outpostv1.TailLogsRequest, stream outpostv1.OutpostService_TailLogsServer) error {
	if _, err := s.store.Get(req.GetId()); err != nil {
		return status.Errorf(codes.NotFound, "run not found: %s", req.GetId())
	}

	logPath := filepath.Join(s.runsDir, req.GetId(), "output.log")

	f, err := openLogFile(stream.Context(), logPath, req.GetFollow())
	if err != nil {
		return err
	}

	offset, err := sendExistingLines(f, stream)
	_ = f.Close()
	if err != nil {
		return err
	}

	if !req.GetFollow() {
		return nil
	}

	return s.followLog(req.GetId(), logPath, offset, stream)
}

type canceler interface {
	Done() <-chan struct{}
}

func openLogFile(ctx canceler, logPath string, follow bool) (*os.File, error) {
	f, err := os.Open(logPath)
	if err == nil {
		return f, nil
	}

	if !follow {
		return nil, status.Errorf(codes.NotFound, "log file not found")
	}

	return waitForFile(ctx, logPath)
}

func sendExistingLines(f *os.File, stream outpostv1.OutpostService_TailLogsServer) (int64, error) {
	reader := bufio.NewReader(f)

	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			text := strings.TrimRight(line, "\n")
			if sendErr := stream.Send(&outpostv1.TailLogsResponse{Line: text}); sendErr != nil {
				return 0, sendErr
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return 0, readErr
			}
			break
		}
	}

	pos, _ := f.Seek(0, io.SeekCurrent)
	return pos, nil
}

func (s *Server) followLog(runID, logPath string, offset int64, stream outpostv1.OutpostService_TailLogsServer) error {
	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			newOffset, err := sendNewLines(logPath, offset, stream)
			if err != nil {
				return err
			}
			offset = newOffset

			st, err := s.store.GetStatus(runID)
			if err != nil {
				return nil
			}
			if st != store.StatusPending && st != store.StatusRunning {
				_, _ = sendNewLines(logPath, offset, stream)
				return nil
			}
		}
	}
}

func sendNewLines(logPath string, offset int64, stream outpostv1.OutpostService_TailLogsServer) (int64, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return offset, nil
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, nil
	}

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			text := strings.TrimRight(line, "\n")
			if sendErr := stream.Send(&outpostv1.TailLogsResponse{Line: text}); sendErr != nil {
				return offset, sendErr
			}
		}
		if err != nil {
			break
		}
	}

	pos, _ := f.Seek(0, io.SeekCurrent)
	return pos, nil
}

func waitForFile(ctx canceler, path string) (*os.File, error) {
	ticker := time.NewTicker(logPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, status.Error(codes.Canceled, "client disconnected")
		case <-ticker.C:
			f, err := os.Open(path)
			if err == nil {
				return f, nil
			}
		}
	}
}
