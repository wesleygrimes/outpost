package cmd

import (
	"encoding/json"
	"os"
	"slices"
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

// printJSON marshals v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
