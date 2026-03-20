package ui

import (
	"strings"
	"testing"
)

//nolint:paralleltest // tests mutate shared globals
func TestChecklistSuccess(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	got := captureStderr(func() {
		cl := NewChecklist("Test")
		cl.Success("Step one")
		cl.Success("Step two")
		cl.Close()
	})

	if !strings.Contains(got, "OUTPOST") {
		t.Error("missing brand header")
	}
	if !strings.Contains(got, "│  ✓ Step one") {
		t.Errorf("missing step one in %q", got)
	}
	if !strings.Contains(got, "│  ✓ Step two") {
		t.Errorf("missing step two in %q", got)
	}
	if !strings.Contains(got, "└") {
		t.Errorf("missing closer in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestChecklistMixedResults(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	var cl *Checklist
	got := captureStderr(func() {
		cl = NewChecklist("Diagnostics")
		cl.Success("Check one passed")
		cl.Fail("Check two failed")
		cl.Success("Check three passed")
		cl.Close()
	})

	if !strings.Contains(got, "✓ Check one passed") {
		t.Errorf("missing first success in %q", got)
	}
	if !strings.Contains(got, "✗ Check two failed") {
		t.Errorf("missing failure in %q", got)
	}
	if !strings.Contains(got, "✓ Check three passed") {
		t.Errorf("missing third success in %q", got)
	}
	if !cl.Failed() {
		t.Error("Failed() should be true")
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestChecklistField(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	got := captureStderr(func() {
		cl := NewChecklist("Detail")
		cl.Field("Branch", "main")
		cl.Close()
	})

	if !strings.Contains(got, "│") {
		t.Errorf("field missing │ prefix in %q", got)
	}
	if !strings.Contains(got, "Branch") {
		t.Errorf("field missing label in %q", got)
	}
	if !strings.Contains(got, "main") {
		t.Errorf("field missing value in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestChecklistCloseWith(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	got := captureStderr(func() {
		cl := NewChecklist("Test")
		cl.Success("Done")
		cl.CloseWith("Run op-7f3a")
	})

	if !strings.Contains(got, "└  Run op-7f3a") {
		t.Errorf("CloseWith missing in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestChecklistErrorAndFix(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	got := captureStderr(func() {
		cl := NewChecklist("Test")
		cl.Fail("Something broke")
		cl.Row("")
		cl.Error("could not connect")
		cl.Fix("outpost login host:9090 token")
		cl.Close()
	})

	if !strings.Contains(got, "Error:") {
		t.Errorf("missing error in %q", got)
	}
	if !strings.Contains(got, "outpost login") {
		t.Errorf("missing fix in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestChecklistFailed(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	Version = testVersion

	_ = captureStderr(func() {
		cl := NewChecklist("Test")
		if cl.Failed() {
			t.Error("Failed() should be false before Fail()")
		}
		cl.Fail("Bad")
		if !cl.Failed() {
			t.Error("Failed() should be true after Fail()")
		}
		cl.Close()
	})
}

//nolint:paralleltest // tests mutate shared globals
func TestHint(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		Hint("Watch", "outpost status op-7f3a --follow")
	})

	if !strings.Contains(got, "Watch") {
		t.Errorf("missing label in %q", got)
	}
	if !strings.Contains(got, "outpost status op-7f3a --follow") {
		t.Errorf("missing command in %q", got)
	}
}
