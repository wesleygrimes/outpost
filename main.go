package main

import (
	"fmt"
	"os"

	"github.com/wesgrimes/outpost/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "setup":
		err = cmd.Setup()
	case "serve":
		err = cmd.Serve()
	case "runs":
		err = cmd.Runs(os.Args[2:])
	case "login":
		err = cmd.Login(os.Args[2:])
	case "handoff":
		err = cmd.Handoff(os.Args[2:])
	case "status":
		err = cmd.Status(os.Args[2:])
	case "pickup":
		err = cmd.Pickup(os.Args[2:])
	case "drop":
		err = cmd.Drop(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: outpost <command>

Server commands:
  setup    Configure a new Outpost server
  serve    Start the Outpost gRPC server
  runs     List runs (server-local)

Client commands:
  login    Connect to an Outpost server
  handoff  Hand off work to the server
  status   Check run status
  pickup   Download completed patch
  drop     Drop a run`)
}
