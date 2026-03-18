package grpcserver

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/store"
)

const testToken = "test-token-abc123"

func setupTestServer(t *testing.T) (outpostv1.OutpostServiceClient, *store.Store) {
	t.Helper()

	st := store.New()
	cfg := &config.ServerConfig{
		Port:              0,
		Token:             testToken,
		MaxConcurrentRuns: 3,
	}

	srv, err := New(cfg, st)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return outpostv1.NewOutpostServiceClient(conn), st
}

func authCtx() context.Context {
	return metadata.AppendToOutgoingContext(
		context.Background(),
		"authorization", "Bearer "+testToken,
	)
}

func TestHealthCheck_NoAuth(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	resp, err := client.HealthCheck(context.Background(), &outpostv1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if resp.GetStatus() != "ok" {
		t.Errorf("status = %q, want %q", resp.GetStatus(), "ok")
	}
}

func TestGetRun_NoAuth_Rejected(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	_, err := client.GetRun(context.Background(), &outpostv1.GetRunRequest{Id: "test"})
	if err == nil {
		t.Fatal("expected error without auth")
	}
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.Unauthenticated {
			t.Errorf("code = %v, want Unauthenticated", s.Code())
		}
	}
}

func TestGetRun_BadToken_Rejected(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	ctx := metadata.AppendToOutgoingContext(
		context.Background(),
		"authorization", "Bearer wrong-token",
	)
	_, err := client.GetRun(ctx, &outpostv1.GetRunRequest{Id: "test"})
	if err == nil {
		t.Fatal("expected error with bad token")
	}
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.Unauthenticated {
			t.Errorf("code = %v, want Unauthenticated", s.Code())
		}
	}
}

func TestGetRun_ValidAuth(t *testing.T) {
	t.Parallel()
	client, st := setupTestServer(t)

	st.Add(&store.Run{
		ID:        "test-run",
		Name:      "my test",
		Mode:      store.ModeHeadless,
		Status:    store.StatusComplete,
		CreatedAt: time.Now(),
	})

	resp, err := client.GetRun(authCtx(), &outpostv1.GetRunRequest{Id: "test-run"})
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}

	run := resp.GetRun()
	if run.GetId() != "test-run" {
		t.Errorf("ID = %q, want %q", run.GetId(), "test-run")
	}
	if run.GetName() != "my test" {
		t.Errorf("Name = %q, want %q", run.GetName(), "my test")
	}
	if run.GetStatus() != outpostv1.RunStatus_RUN_STATUS_COMPLETE {
		t.Errorf("Status = %v, want COMPLETE", run.GetStatus())
	}
}

func TestGetRun_NotFound(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	_, err := client.GetRun(authCtx(), &outpostv1.GetRunRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.NotFound {
			t.Errorf("code = %v, want NotFound", s.Code())
		}
	}
}

func TestListRuns_Empty(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	resp, err := client.ListRuns(authCtx(), &outpostv1.ListRunsRequest{})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(resp.GetRuns()) != 0 {
		t.Errorf("got %d runs, want 0", len(resp.GetRuns()))
	}
}

func TestListRuns_Multiple(t *testing.T) {
	t.Parallel()
	client, st := setupTestServer(t)

	now := time.Now()
	st.Add(&store.Run{ID: "old", CreatedAt: now.Add(-2 * time.Hour)})
	st.Add(&store.Run{ID: "mid", CreatedAt: now.Add(-1 * time.Hour)})
	st.Add(&store.Run{ID: "new", CreatedAt: now})

	resp, err := client.ListRuns(authCtx(), &outpostv1.ListRunsRequest{})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}

	runs := resp.GetRuns()
	if len(runs) != 3 {
		t.Fatalf("got %d runs, want 3", len(runs))
	}
	if runs[0].GetId() != "new" || runs[1].GetId() != "mid" || runs[2].GetId() != "old" {
		t.Errorf("wrong order: %s, %s, %s", runs[0].GetId(), runs[1].GetId(), runs[2].GetId())
	}
}

func TestRemoveRun_RemovesFromStore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		runID  string
		remove func(outpostv1.OutpostServiceClient, string) error
	}{
		{
			name:  "DropRun",
			runID: "to-drop",
			remove: func(c outpostv1.OutpostServiceClient, id string) error {
				resp, err := c.DropRun(authCtx(), &outpostv1.DropRunRequest{Id: id})
				if err != nil {
					return err
				}
				if resp.GetId() != id {
					t.Errorf("ID = %q, want %q", resp.GetId(), id)
				}
				return nil
			},
		},
		{
			name:  "CleanupRun",
			runID: "to-clean",
			remove: func(c outpostv1.OutpostServiceClient, id string) error {
				resp, err := c.CleanupRun(authCtx(), &outpostv1.CleanupRunRequest{Id: id})
				if err != nil {
					return err
				}
				if resp.GetStatus() != "cleaned" {
					t.Errorf("status = %q, want %q", resp.GetStatus(), "cleaned")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, st := setupTestServer(t)

			st.Add(&store.Run{ID: tt.runID, Status: store.StatusComplete, CreatedAt: time.Now()})

			if err := tt.remove(client, tt.runID); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			_, err := client.GetRun(authCtx(), &outpostv1.GetRunRequest{Id: tt.runID})
			if err == nil {
				t.Fatalf("expected NotFound after %s", tt.name)
			}
		})
	}
}

func TestServerDoctor_ReturnsFields(t *testing.T) {
	t.Parallel()
	client, st := setupTestServer(t)

	// Add some runs to verify counts.
	st.Add(&store.Run{ID: "running-1", Status: store.StatusRunning, CreatedAt: time.Now()})
	st.Add(&store.Run{ID: "done-1", Status: store.StatusComplete, CreatedAt: time.Now()})

	resp, err := client.ServerDoctor(authCtx(), &outpostv1.ServerDoctorRequest{})
	if err != nil {
		t.Fatalf("ServerDoctor: %v", err)
	}

	if resp.GetVersion() == "" {
		t.Error("version should not be empty")
	}
	if resp.GetUptime() == "" {
		t.Error("uptime should not be empty")
	}
	if resp.GetActiveRuns() != 1 {
		t.Errorf("active_runs = %d, want 1", resp.GetActiveRuns())
	}
	if resp.GetMaxRuns() != 3 {
		t.Errorf("max_runs = %d, want 3", resp.GetMaxRuns())
	}
	if resp.GetTotalRuns() != 2 {
		t.Errorf("total_runs = %d, want 2", resp.GetTotalRuns())
	}
}

func TestServerDoctor_NoAuth_Rejected(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	_, err := client.ServerDoctor(context.Background(), &outpostv1.ServerDoctorRequest{})
	if err == nil {
		t.Fatal("expected error without auth")
	}
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.Unauthenticated {
			t.Errorf("code = %v, want Unauthenticated", s.Code())
		}
	}
}

func TestHandoff_DataFirst_Rejected(t *testing.T) {
	t.Parallel()
	client, _ := setupTestServer(t)

	stream, err := client.Handoff(authCtx())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Send data as first message (should be metadata).
	err = stream.Send(&outpostv1.HandoffRequest{
		Payload: &outpostv1.HandoffRequest_Data{Data: []byte("hello")},
	})
	if err != nil {
		// Stream send may succeed even if server rejects; the error comes on CloseAndRecv.
		t.Logf("send returned early error: %v", err)
	}

	_, err = stream.CloseAndRecv()
	if err == nil {
		t.Fatal("expected error for data-first handoff")
	}
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.InvalidArgument {
			t.Errorf("code = %v, want InvalidArgument", s.Code())
		}
	}
}

func TestHandoff_InvalidMetadata_Rejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan string
		mode outpostv1.RunMode
	}{
		{
			name: "EmptyPlan",
			plan: "",
			mode: outpostv1.RunMode_RUN_MODE_HEADLESS,
		},
		{
			name: "InvalidMode",
			plan: "do something",
			mode: outpostv1.RunMode_RUN_MODE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, _ := setupTestServer(t)

			stream, err := client.Handoff(authCtx())
			if err != nil {
				t.Fatalf("open stream: %v", err)
			}

			_ = stream.Send(&outpostv1.HandoffRequest{
				Payload: &outpostv1.HandoffRequest_Metadata{
					Metadata: &outpostv1.HandoffMetadata{
						Plan: tt.plan,
						Mode: tt.mode,
					},
				},
			})

			_, err = stream.CloseAndRecv()
			if err == nil {
				t.Fatal("expected error for invalid metadata")
			}
			if s, ok := status.FromError(err); ok {
				if s.Code() != codes.InvalidArgument {
					t.Errorf("code = %v, want InvalidArgument", s.Code())
				}
			}
		})
	}
}

func TestHandoff_AtCapacity(t *testing.T) {
	t.Parallel()
	client, st := setupTestServer(t)

	// Fill capacity (MaxConcurrentRuns=3).
	for i := range 3 {
		st.Add(&store.Run{
			ID:        "active-" + string(rune('a'+i)),
			Status:    store.StatusRunning,
			CreatedAt: time.Now(),
		})
	}

	stream, err := client.Handoff(authCtx())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	_ = stream.Send(&outpostv1.HandoffRequest{
		Payload: &outpostv1.HandoffRequest_Metadata{
			Metadata: &outpostv1.HandoffMetadata{
				Plan: "do something",
				Mode: outpostv1.RunMode_RUN_MODE_HEADLESS,
			},
		},
	})

	_, err = stream.CloseAndRecv()
	if err == nil {
		t.Fatal("expected error at capacity")
	}
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.ResourceExhausted {
			t.Errorf("code = %v, want ResourceExhausted", s.Code())
		}
	}
}
