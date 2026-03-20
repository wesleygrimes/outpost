package ui

import (
	"strings"
	"testing"
)

//nolint:paralleltest // tests mutate shared globals
func TestHeaderWithContext(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	got := captureStderr(func() {
		Header("Provisioning myserver.grimes.pro")
	})

	if !strings.Contains(got, "OUTPOST") {
		t.Errorf("Header() missing brand name, got %q", got)
	}
	if !strings.Contains(got, testVersion) {
		t.Errorf("Header() missing version, got %q", got)
	}
	if !strings.Contains(got, "Provisioning myserver.grimes.pro") {
		t.Errorf("Header() missing context, got %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestHeaderWithoutContext(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	got := captureStderr(func() {
		Header("")
	})

	want := "\n  OUTPOST " + testVersion + "\n"
	if got != want {
		t.Errorf("Header(\"\") = %q, want %q", got, want)
	}
}
