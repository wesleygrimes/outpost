package ui

import (
	"fmt"
	"strings"
	"time"
)

const barWidth = 20

// Progress renders a progress bar on stderr with carriage-return updates.
type Progress struct {
	label    string
	start    time.Time
	rendered bool
}

// NewProgress creates a progress bar with the given label.
func NewProgress(label string) *Progress {
	return &Progress{label: label, start: time.Now()}
}

// Update redraws the progress bar. On non-TTY, this is a no-op.
//
//	⠸ Streaming to server...  ████████████████░░░░ 82%
//
// When total is 0, renders indeterminate mode (spinner only, no bar).
func (p *Progress) Update(current, total int64) {
	if !IsTTY {
		return
	}
	p.rendered = true

	if total <= 0 {
		Errf("\r  %s", Spin(p.label+"..."))
		return
	}

	pct := min(int(current*100/total), 100)
	filled := barWidth * pct / 100
	bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barWidth-filled)
	Errf("\r  %s  %s %d%%", Spin(p.label+"..."), bar, pct)
}

// Done clears the progress line and prints a success summary.
//
//	✓ Streamed (18.3 MB in 2.1s)
//
// If no Update was called (instant completion), just prints the summary.
func (p *Progress) Done(summary string) {
	if p.rendered {
		Errf("\r  %s\r", strings.Repeat(" ", TermWidth()-2))
	}
	Errf("  %s\n", Check(summary))
}

// Elapsed returns the time since the progress bar was created.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.start)
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(b int64) string {
	const mb = 1024 * 1024
	if b < mb {
		return fmt.Sprintf("%.0f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/mb)
}
