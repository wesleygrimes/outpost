package ui

import "testing"

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestSymbolWithColor(t *testing.T) {
	ColorEnabled = true

	tests := []struct {
		name string
		fn   func(string) string
		text string
		sym  string
		code string
	}{
		{"Check", Check, "done", SymCheck, "\033[32m"},
		{"Fail", Fail, "error", SymFail, "\033[31m"},
		{"Warn", Warn, "caution", SymWarn, "\033[93m"},
		{"Spin", Spin, "running", SymSpin, "\033[36m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.text)
			want := tt.code + tt.sym + "\033[0m" + " " + tt.text
			if got != want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.text, got, want)
			}
		})
	}
}

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestSymbolWithoutColor(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	tests := []struct {
		name string
		fn   func(string) string
		text string
		sym  string
	}{
		{"Check", Check, "done", SymCheck},
		{"Fail", Fail, "error", SymFail},
		{"Warn", Warn, "caution", SymWarn},
		{"Spin", Spin, "running", SymSpin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.text)
			want := tt.sym + " " + tt.text
			if got != want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.text, got, want)
			}
		})
	}
}

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestSymbolEmptyText(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := Check("")
	if got != SymCheck {
		t.Errorf("Check(\"\") = %q, want %q", got, SymCheck)
	}
}

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestSymbolDotAndWait(t *testing.T) {
	ColorEnabled = true

	dotResult := Symbol(SymDot, "complete")
	wantDot := "\033[32m" + SymDot + "\033[0m" + " complete"
	if dotResult != wantDot {
		t.Errorf("Symbol(SymDot, \"complete\") = %q, want %q", dotResult, wantDot)
	}

	waitResult := Symbol(SymWait, "pending")
	wantWait := "\033[35m" + SymWait + "\033[0m" + " pending"
	if waitResult != wantWait {
		t.Errorf("Symbol(SymWait, \"pending\") = %q, want %q", waitResult, wantWait)
	}
}
