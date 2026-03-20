package ui

import (
	"strings"
	"testing"
)

//nolint:paralleltest // tests mutate shared globals
func TestLogLineNoColor(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStdout(func() {
		LogLine("op-7f3a", "10:42:17", "claude", "I'll start by reading...")
	})

	if !strings.Contains(got, "op-7f3a") {
		t.Errorf("missing run ID in %q", got)
	}
	if !strings.Contains(got, "10:42:17") {
		t.Errorf("missing timestamp in %q", got)
	}
	if !strings.Contains(got, "claude") {
		t.Errorf("missing source in %q", got)
	}
	if !strings.Contains(got, "I'll start by reading...") {
		t.Errorf("missing content in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestLogLineWithColor(t *testing.T) {
	ColorEnabled = true

	got := captureStdout(func() {
		LogLine("op-7f3a", "10:42:17", "tool", "Read file: CLAUDE.md")
	})

	// Should contain ANSI codes for amber run ID and dim source.
	if !strings.Contains(got, "\033[") {
		t.Errorf("expected ANSI codes in colored output, got %q", got)
	}
	if !strings.Contains(got, "Read file: CLAUDE.md") {
		t.Errorf("missing content in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestLogLinef(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStdout(func() {
		LogLinef("op-7f3a", "10:42:17", "tool", "bash: %s (exit %d)", "rspec", 0)
	})

	if !strings.Contains(got, "bash: rspec (exit 0)") {
		t.Errorf("missing formatted content in %q", got)
	}
}
