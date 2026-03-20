package ui

import (
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

// ansiPattern matches ANSI escape sequences for stripping.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// TermWidth returns the width of the terminal attached to stderr,
// defaulting to 80 when detection fails.
func TermWidth() int {
	w, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// IsTerminal reports whether fd refers to a terminal.
func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// StripAnsi removes all ANSI escape codes from s.
func StripAnsi(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// StringWidth returns the visual display width of s after stripping
// ANSI escape codes. It counts bytes of the stripped string, which is
// correct for ASCII/single-width Unicode code points.
func StringWidth(s string) int {
	return len([]rune(StripAnsi(s)))
}

// PadRight pads s with spaces to the given visual width. If s is
// already at or past width, it is returned unchanged.
func PadRight(s string, width int) string {
	vis := StringWidth(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}
