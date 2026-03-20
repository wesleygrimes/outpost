package ui

import "fmt"

// Checklist renders a branded checklist block to stderr.
//
//	OUTPOST v0.1.0  Context
//
//	│  ✓ Step one
//	│  ✓ Step two
//	│  ✗ Step three
//	│
//	└  Summary
type Checklist struct {
	failed bool
}

// NewChecklist prints the brand header and returns a Checklist.
func NewChecklist(context string) *Checklist {
	Header(context)
	Errln()
	return &Checklist{}
}

// Success prints a green check line.
//
// For sequential operations, callers should check Failed() and stop
// before calling Success() (stop on first failure). For diagnostics
// where all checks run independently, mixing Success/Fail is fine.
func (cl *Checklist) Success(text string) {
	Errf("  │  %s\n", Check(text))
}

// Fail prints a red cross line and marks the checklist as failed.
func (cl *Checklist) Fail(text string) {
	cl.failed = true
	Errf("  │  %s\n", Fail(text))
}

// Row prints a plain text line inside the checklist.
func (cl *Checklist) Row(text string) {
	if text == "" {
		Errln("  │")
	} else {
		Errf("  │  %s\n", text)
	}
}

// Field prints a labeled field inside the checklist.
func (cl *Checklist) Field(label, value string) {
	Errf("  │  %s%s\n", Dim(PadRight(label, 14)), value)
}

// Close prints the closing └ line.
func (cl *Checklist) Close() {
	Errln("  └")
}

// CloseWith prints the closing └ line with a summary.
func (cl *Checklist) CloseWith(text string) {
	Errf("  └  %s\n", text)
}

// Error prints an error message inside the checklist.
func (cl *Checklist) Error(text string) {
	Errf("  │  %s %s\n", Red("Error:"), text)
}

// Fix prints a fix command inside the checklist.
func (cl *Checklist) Fix(text string) {
	Errf("  │  %s\n", Blue(text))
}

// Failed reports whether Fail() has been called.
func (cl *Checklist) Failed() bool {
	return cl.failed
}

// Hint prints a next-step hint after the checklist closer.
//
//	Watch:  outpost status op-7f3a --follow
func Hint(label, command string) {
	Errf("  %s  %s\n", Dim(fmt.Sprintf("%-8s", label)), Blue(command))
}
