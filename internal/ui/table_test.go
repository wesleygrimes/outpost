package ui

import (
	"strings"
	"testing"
)

//nolint:paralleltest // tests mutate shared globals
func TestTableRender(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		tbl := NewTable("ID", "Status", "Age")
		tbl.Row("op-7f3a", "running", "2m")
		tbl.Row("op-a1c2", "done", "18m")
		tbl.Render()
	})

	if !strings.Contains(got, "ID") {
		t.Errorf("missing header in %q", got)
	}
	if !strings.Contains(got, "op-7f3a") {
		t.Errorf("missing first row in %q", got)
	}
	if !strings.Contains(got, "op-a1c2") {
		t.Errorf("missing second row in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestTableFooter(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		tbl := NewTable("ID", "Status")
		tbl.Row("op-7f3a", "running")
		tbl.Footer("2 total", "1 running", "1 done")
		tbl.Render()
	})

	if !strings.Contains(got, "·") {
		t.Errorf("footer missing separator in %q", got)
	}
	if !strings.Contains(got, "2 total") {
		t.Errorf("footer missing items in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestTableAlignment(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		tbl := NewTable("Name", "Value")
		tbl.Row("short", "a")
		tbl.Row(Check("longer text"), "b")
		tbl.Render()
	})

	// Both rows should exist; ANSI-aware alignment means the colored
	// row doesn't push columns further than needed.
	if !strings.Contains(got, "short") {
		t.Errorf("missing plain row in %q", got)
	}
	if !strings.Contains(got, "longer text") {
		t.Errorf("missing colored row in %q", got)
	}
}

//nolint:paralleltest // tests mutate shared globals
func TestTableEmpty(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := captureStderr(func() {
		tbl := NewTable("ID", "Status")
		tbl.Render()
	})

	// Should still render headers even with no rows.
	if !strings.Contains(got, "ID") {
		t.Errorf("empty table missing headers in %q", got)
	}
}
