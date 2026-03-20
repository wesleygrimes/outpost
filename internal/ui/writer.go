package ui

import (
	"fmt"
	"os"
)

// Errf writes formatted output to stderr. Output is suppressed in QuietMode.
func Errf(format string, args ...any) {
	if QuietMode {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

// Errln writes a line to stderr. Output is suppressed in QuietMode.
func Errln(args ...any) {
	if QuietMode {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, args...)
}

// Outf writes formatted output to stdout (for machine-readable data).
func Outf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format, args...)
}

// Outln writes a line to stdout (for machine-readable data).
func Outln(args ...any) {
	_, _ = fmt.Fprintln(os.Stdout, args...)
}
