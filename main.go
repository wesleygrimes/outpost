package main

import (
	"fmt"
	"os"

	"github.com/wesgrimes/outpost/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: outpost <setup|serve|runs>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		cmd.Setup()
	case "serve":
		cmd.Serve()
	case "runs":
		cmd.Runs(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
