// Package grpcserver implements the Outpost gRPC service.
package grpcserver

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/runner"
	"github.com/wesgrimes/outpost/internal/store"
)

// Server implements the OutpostService gRPC server.
type Server struct {
	outpostv1.UnimplementedOutpostServiceServer

	cfg       *config.ServerConfig
	store     *store.Store
	grpc      *grpc.Server
	runsDir   string
	registry  *runner.Registry
	startTime time.Time
}

// New creates a gRPC server with TLS (if configured) and auth interceptors.
func New(cfg *config.ServerConfig, st *store.Store) (*Server, error) {
	s := &Server{
		cfg:       cfg,
		store:     st,
		runsDir:   config.RunsDir(),
		registry:  runner.NewRegistry(),
		startTime: time.Now(),
	}

	var opts []grpc.ServerOption

	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		tlsCfg, err := buildTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("tls config: %w", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	opts = append(opts,
		grpc.UnaryInterceptor(s.unaryAuthInterceptor),
		grpc.StreamInterceptor(s.streamAuthInterceptor),
	)

	s.grpc = grpc.NewServer(opts...)
	outpostv1.RegisterOutpostServiceServer(s.grpc, s)

	return s, nil
}

// ListenAndServe starts the gRPC server on the configured port.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	fmt.Fprintf(os.Stderr, "outpost: listening on %s\n", addr)
	return s.grpc.Serve(lis)
}

// Serve starts the gRPC server on the given listener.
func (s *Server) Serve(lis net.Listener) error {
	return s.grpc.Serve(lis)
}

// GracefulStop shuts down the gRPC server gracefully.
func (s *Server) GracefulStop() {
	s.grpc.GracefulStop()
}

const healthCheckMethod = "/outpost.v1.OutpostService/HealthCheck"

func (s *Server) unaryAuthInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	if info.FullMethod == healthCheckMethod {
		return handler(ctx, req)
	}
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (s *Server) streamAuthInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if info.FullMethod == healthCheckMethod {
		return handler(srv, ss)
	}
	if err := s.checkAuth(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

func (s *Server) checkAuth(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	vals := md.Get("authorization")
	if len(vals) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization")
	}

	token := vals[0]
	const bearerPrefix = "Bearer "
	if len(token) > len(bearerPrefix) && token[:len(bearerPrefix)] == bearerPrefix {
		token = token[len(bearerPrefix):]
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
		return status.Error(codes.Unauthenticated, "invalid token")
	}

	return nil
}

func buildTLSConfig(cfg *config.ServerConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("load keypair: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if cfg.TLSCA != "" {
		caCert, err := os.ReadFile(cfg.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("invalid CA certificate")
		}
		tlsCfg.ClientCAs = pool
	}

	return tlsCfg, nil
}

// removeRunData removes a run's directory and deletes it from the store.
func (s *Server) removeRunData(r *store.Run) {
	if r.Dir != "" {
		_ = os.RemoveAll(r.Dir)
	}
	s.store.Delete(r.ID)
}
