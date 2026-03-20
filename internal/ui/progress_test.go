package ui

import (
	"strings"
	"testing"
	"time"
)

//nolint:paralleltest // tests mutate shared globals
func TestProgressDoneWithoutUpdate(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		pb := NewProgress("Uploading")
		pb.Done("Uploaded (5.0 MB in 1.2s)")
	})

	if !strings.Contains(got, "✓") {
		t.Errorf("missing check mark in %q", got)
	}
	if !strings.Contains(got, "Uploaded") {
		t.Errorf("missing summary in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestProgressUpdateNoTTY(t *testing.T) {
	ColorEnabled = false
	origTTY := IsTTY
	IsTTY = false
	defer func() {
		ColorEnabled = true
		IsTTY = origTTY
	}()

	got := captureStderr(func() {
		pb := NewProgress("Streaming")
		pb.Update(50, 100)
		pb.Update(100, 100)
		pb.Done("Streamed")
	})

	// No-TTY: Update is a no-op, only Done prints.
	if strings.Contains(got, "░") || strings.Contains(got, "█") {
		t.Errorf("progress bar should not render on non-TTY, got %q", got)
	}
	if !strings.Contains(got, "Streamed") {
		t.Errorf("missing done summary in %q", got)
	}
}

func TestProgressElapsed(t *testing.T) {
	t.Parallel()

	pb := NewProgress("Test")
	time.Sleep(10 * time.Millisecond)

	if pb.Elapsed() < 10*time.Millisecond {
		t.Errorf("Elapsed() = %v, want >= 10ms", pb.Elapsed())
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bytes int64
		want  string
	}{
		{512 * 1024, "512 KB"},
		{1024 * 1024, "1.0 MB"},
		{18 * 1024 * 1024, "18.0 MB"},
		{1536 * 1024, "1.5 MB"},
	}

	for _, tt := range tests {
		if got := FormatBytes(tt.bytes); got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
