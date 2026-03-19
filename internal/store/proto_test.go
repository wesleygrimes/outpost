package store

import (
	"testing"
	"time"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
)

func TestRunToProto_RoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Microsecond) // proto timestamps have microsecond precision
	finished := now.Add(5 * time.Minute)

	original := &Run{
		ID:              "test-123",
		Name:            "my run",
		Mode:            ModeHeadless,
		Status:          StatusComplete,
		BaseSHA:         "abc123",
		FinalSHA:        "def456",
		CreatedAt:       now,
		FinishedAt:      &finished,
		Attach:          "ssh -t host tmux attach",
		LogTail:         "last line",
		PatchReady:      true,
		Branch:          "fix/auth",
		MaxTurns:        25,
		Subdir:          "packages/core",
		SessionID:       "sess-abc-123",
		ForkedSessionID: "sess-def-456",
		SessionReady:    true,
	}

	proto := RunToProto(original)
	roundTripped := ProtoToRun(proto)

	if roundTripped.ID != original.ID {
		t.Errorf("ID: got %q, want %q", roundTripped.ID, original.ID)
	}
	if roundTripped.Mode != original.Mode {
		t.Errorf("Mode: got %q, want %q", roundTripped.Mode, original.Mode)
	}
	if roundTripped.Status != original.Status {
		t.Errorf("Status: got %q, want %q", roundTripped.Status, original.Status)
	}
	if roundTripped.BaseSHA != original.BaseSHA {
		t.Errorf("BaseSHA: got %q, want %q", roundTripped.BaseSHA, original.BaseSHA)
	}
	if roundTripped.FinalSHA != original.FinalSHA {
		t.Errorf("FinalSHA: got %q, want %q", roundTripped.FinalSHA, original.FinalSHA)
	}
	if roundTripped.PatchReady != original.PatchReady {
		t.Errorf("PatchReady: got %v, want %v", roundTripped.PatchReady, original.PatchReady)
	}
	if roundTripped.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", roundTripped.Branch, original.Branch)
	}
	if roundTripped.MaxTurns != original.MaxTurns {
		t.Errorf("MaxTurns: got %d, want %d", roundTripped.MaxTurns, original.MaxTurns)
	}
	if roundTripped.Subdir != original.Subdir {
		t.Errorf("Subdir: got %q, want %q", roundTripped.Subdir, original.Subdir)
	}
	if roundTripped.SessionID != original.SessionID {
		t.Errorf("SessionID: got %q, want %q", roundTripped.SessionID, original.SessionID)
	}
	if roundTripped.ForkedSessionID != original.ForkedSessionID {
		t.Errorf("ForkedSessionID: got %q, want %q", roundTripped.ForkedSessionID, original.ForkedSessionID)
	}
	if roundTripped.SessionReady != original.SessionReady {
		t.Errorf("SessionReady: got %v, want %v", roundTripped.SessionReady, original.SessionReady)
	}
	if roundTripped.FinishedAt == nil {
		t.Fatal("FinishedAt is nil after round-trip")
	}
	if !roundTripped.FinishedAt.Equal(finished) {
		t.Errorf("FinishedAt: got %v, want %v", roundTripped.FinishedAt, finished)
	}
}

func TestRunToProto_NilFinishedAt(t *testing.T) {
	t.Parallel()
	r := &Run{
		ID:        "test-1",
		Status:    StatusRunning,
		CreatedAt: time.Now(),
	}

	proto := RunToProto(r)
	if proto.GetFinishedAt() != nil {
		t.Error("expected nil FinishedAt in proto")
	}

	back := ProtoToRun(proto)
	if back.FinishedAt != nil {
		t.Error("expected nil FinishedAt after round-trip")
	}
}

func TestRunToProto_Nil(t *testing.T) {
	t.Parallel()
	if RunToProto(nil) != nil {
		t.Error("RunToProto(nil) should return nil")
	}
	if ProtoToRun(nil) != nil {
		t.Error("ProtoToRun(nil) should return nil")
	}
}

func TestStatusToProto_AllValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status Status
		want   outpostv1.RunStatus
	}{
		{StatusPending, outpostv1.RunStatus_RUN_STATUS_PENDING},
		{StatusRunning, outpostv1.RunStatus_RUN_STATUS_RUNNING},
		{StatusComplete, outpostv1.RunStatus_RUN_STATUS_COMPLETE},
		{StatusFailed, outpostv1.RunStatus_RUN_STATUS_FAILED},
		{StatusDropped, outpostv1.RunStatus_RUN_STATUS_DROPPED},
		{"", outpostv1.RunStatus_RUN_STATUS_UNSPECIFIED},
		{"bogus", outpostv1.RunStatus_RUN_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		got := StatusToProto(tt.status)
		if got != tt.want {
			t.Errorf("StatusToProto(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestStatusFromProto_AllValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		proto outpostv1.RunStatus
		want  Status
	}{
		{outpostv1.RunStatus_RUN_STATUS_PENDING, StatusPending},
		{outpostv1.RunStatus_RUN_STATUS_RUNNING, StatusRunning},
		{outpostv1.RunStatus_RUN_STATUS_COMPLETE, StatusComplete},
		{outpostv1.RunStatus_RUN_STATUS_FAILED, StatusFailed},
		{outpostv1.RunStatus_RUN_STATUS_DROPPED, StatusDropped},
		{outpostv1.RunStatus_RUN_STATUS_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		got := StatusFromProto(tt.proto)
		if got != tt.want {
			t.Errorf("StatusFromProto(%v) = %q, want %q", tt.proto, got, tt.want)
		}
	}
}

func TestModeToProto_AllValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		mode Mode
		want outpostv1.RunMode
	}{
		{ModeInteractive, outpostv1.RunMode_RUN_MODE_INTERACTIVE},
		{ModeHeadless, outpostv1.RunMode_RUN_MODE_HEADLESS},
		{"", outpostv1.RunMode_RUN_MODE_UNSPECIFIED},
		{"bogus", outpostv1.RunMode_RUN_MODE_UNSPECIFIED},
	}

	for _, tt := range tests {
		got := ModeToProto(tt.mode)
		if got != tt.want {
			t.Errorf("ModeToProto(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestModeFromProto_AllValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		proto outpostv1.RunMode
		want  Mode
	}{
		{outpostv1.RunMode_RUN_MODE_INTERACTIVE, ModeInteractive},
		{outpostv1.RunMode_RUN_MODE_HEADLESS, ModeHeadless},
		{outpostv1.RunMode_RUN_MODE_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		got := ModeFromProto(tt.proto)
		if got != tt.want {
			t.Errorf("ModeFromProto(%v) = %q, want %q", tt.proto, got, tt.want)
		}
	}
}
