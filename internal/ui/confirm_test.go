package ui

import "testing"

//nolint:paralleltest // tests mutate shared globals
func TestConfirmForceMode(t *testing.T) {
	orig := ForceMode
	ForceMode = true
	defer func() { ForceMode = orig }()

	got := Confirm("Test?", "Yes", "No")
	if got != 0 {
		t.Errorf("Confirm() with ForceMode = %d, want 0", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestConfirmNoTTY(t *testing.T) {
	origTTY := IsTTY
	origForce := ForceMode
	origColor := ColorEnabled
	IsTTY = false
	ForceMode = false
	ColorEnabled = false
	defer func() {
		IsTTY = origTTY
		ForceMode = origForce
		ColorEnabled = origColor
	}()

	got := captureStderr(func() {
		result := Confirm("Test?", "Yes", "No")
		if result != -1 {
			t.Errorf("Confirm() on non-TTY = %d, want -1", result)
		}
	})

	if got == "" {
		t.Error("expected warning message on non-TTY")
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestConfirmNoOptions(t *testing.T) {
	got := Confirm("Test?")
	if got != -1 {
		t.Errorf("Confirm() with no options = %d, want -1", got)
	}
}
