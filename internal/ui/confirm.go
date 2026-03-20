package ui

import (
	"os"

	"golang.org/x/term"
)

// Confirm displays an interactive selection prompt and returns the chosen index.
// Returns -1 if cancelled (Ctrl+C, Escape, or non-TTY without ForceMode).
// If ForceMode is true, returns 0 (first option) without prompting.
//
//	Include uncommitted changes?
//	● Yes, include working tree as-is
//	  No, only send committed files
//	  Cancel
func Confirm(prompt string, options ...string) int {
	if ForceMode {
		return 0
	}
	if !IsTTY {
		Errln("  " + Warn("Cannot prompt: not a terminal. Use --force."))
		return -1
	}
	if len(options) == 0 {
		return -1
	}

	selected := 0

	// Switch stdin to raw mode for single-keypress reading.
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return -1
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	renderOptions(prompt, options, selected)

	for {
		key := readKey(fd)
		switch key {
		case keyEnter:
			clearLines(len(options) + 1) // prompt + options
			Errf("  %s %s\n", Amber("\u25cf"), options[selected])
			return selected
		case keyUp:
			if selected > 0 {
				selected--
				clearLines(len(options) + 1)
				renderOptions(prompt, options, selected)
			}
		case keyDown:
			if selected < len(options)-1 {
				selected++
				clearLines(len(options) + 1)
				renderOptions(prompt, options, selected)
			}
		case keyEsc, keyCtrlC:
			clearLines(len(options) + 1)
			return -1
		}
	}
}

const (
	keyEnter = iota + 1
	keyUp
	keyDown
	keyEsc
	keyCtrlC
	keyUnknown
)

func readKey(fd int) int {
	buf := make([]byte, 3)
	n, err := os.NewFile(uintptr(fd), "/dev/stdin").Read(buf)
	if err != nil || n == 0 {
		return keyUnknown
	}

	switch {
	case n == 1 && (buf[0] == '\r' || buf[0] == '\n'):
		return keyEnter
	case n == 1 && buf[0] == 3: // Ctrl+C
		return keyCtrlC
	case n == 1 && buf[0] == 27: // standalone Escape
		return keyEsc
	case n >= 3 && buf[0] == 27 && buf[1] == '[':
		switch buf[2] {
		case 'A':
			return keyUp
		case 'B':
			return keyDown
		}
	}
	return keyUnknown
}

func renderOptions(prompt string, options []string, selected int) {
	Errf("  %s\n", prompt)
	for i, opt := range options {
		if i == selected {
			Errf("  %s %s\n", Amber("\u25cf"), opt)
		} else {
			Errf("  %s\n", Dim("  "+opt))
		}
	}
}

// clearLines moves cursor up n lines and clears each one.
func clearLines(n int) {
	for range n {
		Errf("\033[A\033[2K")
	}
}
