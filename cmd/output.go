package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/wesgrimes/outpost/internal/config"
)

// hasFlag checks for a boolean flag in args and returns cleaned args.
func hasFlag(args []string, flags ...string) (found bool, cleaned []string) {
	cleaned = make([]string, 0, len(args))

	for _, a := range args {
		if slices.Contains(flags, a) {
			found = true
		} else {
			cleaned = append(cleaned, a)
		}
	}

	return found, cleaned
}

// printHeader prints the unified Outpost header with version and server.
func printHeader() {
	cfg, err := config.LoadClient()
	server := ""
	if err == nil {
		server = cfg.Server
	}

	header := "Outpost " + Version
	if server != "" {
		header += "    " + server
	}
	fmt.Println(header)
	fmt.Println(strings.Repeat("\u2500", len(header)))
}

// printJSON marshals v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newTable returns a tabwriter configured for aligned column output.
func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// printField prints a labeled field with consistent alignment.
func printField(label, value string) {
	fmt.Printf("  %-16s%s\n", label, value)
}
