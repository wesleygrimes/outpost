package ui

import "fmt"

// LogLine prints a structured log line to stdout.
//
// Log lines go to stdout (greppable data), not stderr.
// Color is applied only when stdout is a terminal.
//
//	op-7f3a 10:42:17 claude I'll start by reading...
//	op-7f3a 10:42:17 tool   Read file: CLAUDE.md
func LogLine(runID, timestamp, source, content string) {
	coloredSource := source
	if ColorEnabled {
		switch source {
		case "claude":
			coloredSource = Purple(source)
		default:
			coloredSource = Dim(source)
		}
	}

	Outf("%-8s%-10s%-8s%s\n",
		Amber(runID),
		Dim(timestamp),
		coloredSource,
		content,
	)
}

// LogLinef is a convenience wrapper around LogLine with a formatted content string.
func LogLinef(runID, timestamp, source, format string, args ...any) {
	LogLine(runID, timestamp, source, fmt.Sprintf(format, args...))
}
