package ui

import "testing"

func TestStripAnsi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello", "hello"},
		{"single code", "\033[31mred\033[0m", "red"},
		{"nested codes", "\033[1m\033[31mbold red\033[0m", "bold red"},
		{"empty", "", ""},
		{"no escape", "plain text here", "plain text here"},
		{"multiple colors", "\033[32mgreen\033[0m and \033[34mblue\033[0m", "green and blue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := StripAnsi(tt.input)
			if got != tt.want {
				t.Errorf("StripAnsi(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain ascii", "hello", 5},
		{"with ansi", "\033[31mhello\033[0m", 5},
		{"empty", "", 0},
		{"symbol", "\u2713", 1},
		{"colored symbol", "\033[32m\u2713\033[0m done", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := StringWidth(tt.input)
			if got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		width int
		want  int // expected visual width of result
	}{
		{"pad short", "hi", 10, 10},
		{"already wide", "hello world", 5, 11},
		{"exact width", "hello", 5, 5},
		{"with ansi", "\033[31mhi\033[0m", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := PadRight(tt.input, tt.width)
			gotWidth := StringWidth(got)
			if gotWidth != tt.want {
				t.Errorf("PadRight(%q, %d) visual width = %d, want %d",
					tt.input, tt.width, gotWidth, tt.want)
			}
		})
	}
}

func TestTermWidthDefault(t *testing.T) {
	t.Parallel()

	// In a test environment, TermWidth should return 80 (the default)
	// since stderr is not a terminal.
	w := TermWidth()
	if w <= 0 {
		t.Errorf("TermWidth() = %d, want > 0", w)
	}
}
