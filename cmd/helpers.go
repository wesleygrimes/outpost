package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const boxWidth = 70

func logClose(c io.Closer) {
	if err := c.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close: %v\n", err)
	}
}

// Box-drawing helpers for polished CLI output.

func printBoxTop(title, subtitle string) {
	label := title
	if subtitle != "" {
		label = title + " \u2500\u2500 " + subtitle
	}
	// Top: ╭─ <label> ─...─╮
	pad := max(boxWidth-4-len(label), 1)
	fmt.Fprintf(os.Stderr, "\u256d\u2500 %s %s\u256e\n", label, strings.Repeat("\u2500", pad))
	fmt.Fprintf(os.Stderr, "\u2502%s\u2502\n", strings.Repeat(" ", boxWidth-2))
}

func printBoxBottom() {
	fmt.Fprintf(os.Stderr, "\u2502%s\u2502\n", strings.Repeat(" ", boxWidth-2))
	fmt.Fprintf(os.Stderr, "\u2570%s\u256f\n", strings.Repeat("\u2500", boxWidth-2))
}

func printBoxDivider(label string) {
	pad := max(boxWidth-4-len(label), 1)
	fmt.Fprintf(os.Stderr, "\u251c\u2500 %s %s\u2524\n", label, strings.Repeat("\u2500", pad))
}

func printBoxRow(text string) {
	padded := text
	visible := len(text)
	if visible < boxWidth-4 {
		padded = text + strings.Repeat(" ", boxWidth-4-visible)
	}
	fmt.Fprintf(os.Stderr, "\u2502  %s\u2502\n", padded)
}

func printCheckItem(label, value string) {
	printBoxRow(fmt.Sprintf("\u2713  %-16s %s", label, value))
}

func printFailItem(label, value string) {
	printBoxRow(fmt.Sprintf("\u2717  %-16s %s", label, value))
}
