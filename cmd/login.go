package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Login saves Outpost server credentials to ~/.outpost-url and ~/.outpost-token.
func Login(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: outpost login <url> <token>")
		fmt.Fprintln(os.Stderr, "  url    Outpost server URL (e.g. http://10.0.0.5:7600)")
		fmt.Fprintln(os.Stderr, "  token  Bearer token from 'outpost setup'")
		os.Exit(1)
	}

	url := strings.TrimRight(args[0], "/")
	token := strings.TrimSpace(args[1])

	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("finding home directory: %v", err)
	}

	urlPath := filepath.Join(home, ".outpost-url")
	tokenPath := filepath.Join(home, ".outpost-token")

	if err := os.WriteFile(urlPath, []byte(url+"\n"), 0o600); err != nil {
		fatalf("writing %s: %v", urlPath, err)
	}

	if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0o600); err != nil {
		fatalf("writing %s: %v", tokenPath, err)
	}

	fmt.Printf("Credentials saved to %s and %s\n", urlPath, tokenPath)
}
