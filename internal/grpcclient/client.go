// Package grpcclient provides a client for the Outpost gRPC service.
package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	outpostv1 "github.com/wesleygrimes/outpost/gen/outpost/v1"
	"github.com/wesleygrimes/outpost/internal/config"
	"github.com/wesleygrimes/outpost/internal/store"
)

const handoffChunkSize = 64 * 1024 // 64 KiB

// Client wraps a gRPC connection to an Outpost server.
type Client struct {
	conn  *grpc.ClientConn
	svc   outpostv1.OutpostServiceClient
	token string
}

// HandoffMeta holds metadata for a handoff request.
type HandoffMeta struct {
	SessionID    string
	SessionJSONL []byte
	Mode         outpostv1.RunMode
	Name         string
	Branch       string
	MaxTurns     int32
	Subdir       string
}

// HandoffResult holds the response from a handoff.
type HandoffResult struct {
	ID          string
	Status      outpostv1.RunStatus
	Attach      string
	AttachLocal string
}

// ServerDoctorResult holds the response from a server doctor check.
type ServerDoctorResult struct {
	Version         string
	Uptime          string
	DiskFree        string
	ClaudeInstalled bool
	TmuxInstalled   bool
	ActiveRuns      int32
	MaxRuns         int32
	TotalRuns       int32
}

// New dials the target with the given token and options.
func New(target, token string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	return &Client{
		conn:  conn,
		svc:   outpostv1.NewOutpostServiceClient(conn),
		token: token,
	}, nil
}

// Load reads credentials from ~/.config/outpost/config.yaml and creates a Client.
func Load() (*Client, error) {
	cfg, err := config.LoadClient()
	if err != nil {
		return nil, err
	}

	dialOpt, err := tlsDialOption(cfg.CACert)
	if err != nil {
		return nil, err
	}

	return New(cfg.Server, cfg.Token, dialOpt, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(64<<20)))
}

// TLSDialOption returns a gRPC dial option for TLS using the given CA cert path.
// Exported for use by server-local commands (e.g. runs).
func TLSDialOption(caPath string) (grpc.DialOption, error) {
	return tlsDialOption(caPath)
}

// tlsDialOption returns a gRPC dial option for TLS.
// If caPath is set and the file exists, uses that CA.
// Otherwise uses system TLS (works behind Traefik with real certs).
func tlsDialOption(caPath string) (grpc.DialOption, error) {
	pool, err := loadCAPool(caPath)
	if err != nil {
		return nil, err
	}
	// pool may be nil, which means use system roots.
	return grpc.WithTransportCredentials(
		credentials.NewTLS(&tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		}),
	), nil
}

// loadCAPool loads a custom CA pool from the given path.
// Returns nil pool (use system roots) if caPath is empty or the file doesn't exist.
func loadCAPool(caPath string) (*x509.CertPool, error) {
	if caPath == "" {
		return nil, nil //nolint:nilnil // nil pool means use system roots
	}
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil // missing file means use system roots
		}
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("invalid CA certificate")
	}
	return pool, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// HealthCheck calls the HealthCheck RPC.
func (c *Client) HealthCheck(ctx context.Context) (string, error) {
	resp, err := c.svc.HealthCheck(ctx, &outpostv1.HealthCheckRequest{})
	if err != nil {
		return "", err
	}
	return resp.GetStatus(), nil
}

// ServerDoctor calls the ServerDoctor RPC.
func (c *Client) ServerDoctor(ctx context.Context) (*ServerDoctorResult, error) {
	resp, err := c.svc.ServerDoctor(c.authCtx(ctx), &outpostv1.ServerDoctorRequest{})
	if err != nil {
		return nil, err
	}
	return &ServerDoctorResult{
		Version:         resp.GetVersion(),
		Uptime:          resp.GetUptime(),
		DiskFree:        resp.GetDiskFree(),
		ClaudeInstalled: resp.GetClaudeInstalled(),
		TmuxInstalled:   resp.GetTmuxInstalled(),
		ActiveRuns:      resp.GetActiveRuns(),
		MaxRuns:         resp.GetMaxRuns(),
		TotalRuns:       resp.GetTotalRuns(),
	}, nil
}

// GetRun fetches a single run by ID.
func (c *Client) GetRun(ctx context.Context, id string) (*store.Run, error) {
	resp, err := c.svc.GetRun(c.authCtx(ctx), &outpostv1.GetRunRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return store.ProtoToRun(resp.GetRun()), nil
}

// ListRuns returns all runs.
func (c *Client) ListRuns(ctx context.Context) ([]*store.Run, error) {
	resp, err := c.svc.ListRuns(c.authCtx(ctx), &outpostv1.ListRunsRequest{})
	if err != nil {
		return nil, err
	}
	runs := make([]*store.Run, 0, len(resp.GetRuns()))
	for _, pr := range resp.GetRuns() {
		runs = append(runs, store.ProtoToRun(pr))
	}
	return runs, nil
}

// ConvertMode changes a running session between interactive and headless.
func (c *Client) ConvertMode(ctx context.Context, id string, targetMode outpostv1.RunMode) (*store.Run, error) {
	resp, err := c.svc.ConvertMode(c.authCtx(ctx), &outpostv1.ConvertModeRequest{
		Id:         id,
		TargetMode: targetMode,
	})
	if err != nil {
		return nil, err
	}
	return store.ProtoToRun(resp.GetRun()), nil
}

// DropRun stops and discards a run.
func (c *Client) DropRun(ctx context.Context, id string) (string, error) {
	resp, err := c.svc.DropRun(c.authCtx(ctx), &outpostv1.DropRunRequest{Id: id})
	if err != nil {
		return "", err
	}
	return resp.GetId(), nil
}

// CleanupRun removes a run's data.
func (c *Client) CleanupRun(ctx context.Context, id string) error {
	_, err := c.svc.CleanupRun(c.authCtx(ctx), &outpostv1.CleanupRunRequest{Id: id})
	return err
}

// Handoff streams an archive to the server with metadata.
func (c *Client) Handoff(ctx context.Context, archivePath string, meta *HandoffMeta, onProgress func(sent, total int64)) (*HandoffResult, error) {
	stream, err := c.svc.Handoff(c.authCtx(ctx))
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	if err := stream.Send(&outpostv1.HandoffRequest{
		Payload: &outpostv1.HandoffRequest_Metadata{
			Metadata: &outpostv1.HandoffMetadata{
				SessionId:    meta.SessionID,
				SessionJsonl: meta.SessionJSONL,
				Mode:         meta.Mode,
				Name:         meta.Name,
				Branch:       meta.Branch,
				MaxTurns:     meta.MaxTurns,
				Subdir:       meta.Subdir,
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("send metadata: %w", err)
	}

	if err := sendArchiveChunks(stream, archivePath, onProgress); err != nil {
		return nil, err
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}

	return &HandoffResult{
		ID:          resp.GetId(),
		Status:      resp.GetStatus(),
		Attach:      resp.GetAttach(),
		AttachLocal: resp.GetAttachLocal(),
	}, nil
}

// TailLogs streams log lines. The caller reads from the returned stream.
func (c *Client) TailLogs(ctx context.Context, id string, follow bool) (outpostv1.OutpostService_TailLogsClient, error) {
	return c.svc.TailLogs(c.authCtx(ctx), &outpostv1.TailLogsRequest{
		Id:     id,
		Follow: follow,
	})
}

// downloadToFile writes streamed byte chunks to destPath.
// recvFunc should call Recv on the stream and return the data bytes.
func downloadToFile(destPath string, recvFunc func() ([]byte, error)) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	var writeErr error
	defer func() {
		if err := f.Close(); err != nil && writeErr == nil {
			writeErr = err
		}
	}()

	for {
		data, recvErr := recvFunc()
		if errors.Is(recvErr, io.EOF) {
			return writeErr
		}
		if recvErr != nil {
			return fmt.Errorf("recv chunk: %w", recvErr)
		}
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("write chunk: %w", err)
		}
	}
}

// DownloadPatch downloads a patch to destPath.
func (c *Client) DownloadPatch(ctx context.Context, id, destPath string) error {
	stream, err := c.svc.DownloadPatch(c.authCtx(ctx), &outpostv1.DownloadPatchRequest{Id: id})
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	return downloadToFile(destPath, func() ([]byte, error) {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return chunk.GetData(), nil
	})
}

// DownloadSession downloads a forked session JSONL to destPath.
func (c *Client) DownloadSession(ctx context.Context, id, destPath string) error {
	stream, err := c.svc.DownloadSession(c.authCtx(ctx), &outpostv1.DownloadSessionRequest{Id: id})
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	return downloadToFile(destPath, func() ([]byte, error) {
		chunk, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return chunk.GetData(), nil
	})
}

func (c *Client) authCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token)
}

func sendArchiveChunks(stream outpostv1.OutpostService_HandoffClient, archivePath string, onProgress func(sent, total int64)) error {
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}
	total := info.Size()

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, handoffChunkSize)
	var sent int64

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&outpostv1.HandoffRequest{
				Payload: &outpostv1.HandoffRequest_Data{Data: buf[:n]},
			}); sendErr != nil {
				return fmt.Errorf("send chunk: %w", sendErr)
			}
			sent += int64(n)
			if onProgress != nil {
				onProgress(sent, total)
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("read archive: %w", readErr)
		}
	}
}
