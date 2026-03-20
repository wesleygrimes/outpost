package ui

import (
	"strings"
	"testing"
)

//nolint:paralleltest // tests mutate shared globals
func TestField(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		Field("Branch", "main")
	})

	if !strings.Contains(got, "Branch") {
		t.Errorf("Field missing label in %q", got)
	}
	if !strings.Contains(got, "main") {
		t.Errorf("Field missing value in %q", got)
	}
}
