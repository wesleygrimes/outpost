// Package ui provides the Outpost CLI design system: colors, symbols,
// terminal utilities, and output writers.
package ui

import "os"

// Global output mode flags, set once at startup.
//
//nolint:gochecknoglobals // UI state is inherently global for a CLI
var (
	// ColorEnabled controls whether ANSI color codes are emitted.
	ColorEnabled bool

	// QuietMode suppresses all human chrome on stderr.
	QuietMode bool

	// ForceMode skips confirmation prompts.
	ForceMode bool

	// IsTTY is true when stderr is a terminal.
	IsTTY bool

	// Version holds the Outpost version string for headers.
	Version string
)

// Init sets up global UI state. Call once at program start.
func Init(version string) {
	Version = version
	IsTTY = IsTerminal(os.Stderr.Fd())
	ColorEnabled = IsTTY

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		ColorEnabled = false
	}
}

// SetColor overrides color detection (e.g. from --no-color flag).
func SetColor(enabled bool) {
	ColorEnabled = enabled
}
