package ui

import (
	"bytes"
	"os"
)

const testVersion = "test"

// captureStderr runs fn and returns everything written to os.Stderr.
func captureStderr(fn func()) string {
	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// captureStdout runs fn and returns everything written to os.Stdout.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}
