package main

import (
	"fmt"
	"os"

	"github.com/wesgrimes/outpost/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: outpost <command>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Server:  setup, serve, runs")
		fmt.Fprintln(os.Stderr, "Client:  login, handoff, status, pickup, kill")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		cmd.Setup()
	case "serve":
		cmd.Serve()
	case "runs":
		cmd.Runs(os.Args[2:])
	case "login":
		cmd.Login(os.Args[2:])
	case "handoff":
		cmd.Handoff(os.Args[2:])
	case "status":
		cmd.Status(os.Args[2:])
	case "pickup":
		cmd.Pickup(os.Args[2:])
	case "kill":
		cmd.KillRun(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
