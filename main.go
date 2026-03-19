package main

import (
	"fmt"
	"os"

	"github.com/wesgrimes/outpost/cmd"
)

// version is set by -ldflags at build time.
var version = "dev"

func main() {
	// Inject version into cmd package.
	cmd.Version = version

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	// Server commands.
	case "server":
		err = cmd.Server(os.Args[2:])
	case "serve":
		err = cmd.Serve()

	// Client commands.
	case "login":
		err = cmd.Login(os.Args[2:])
	case "doctor":
		err = cmd.Doctor()
	case "handoff":
		err = cmd.Handoff(os.Args[2:])
	case "status":
		err = cmd.Status(os.Args[2:])
	case "logs":
		err = cmd.Logs(os.Args[2:])
	case "pickup":
		err = cmd.Pickup(os.Args[2:])
	case "drop":
		err = cmd.Drop(os.Args[2:])
	case "convert":
		err = cmd.Convert(os.Args[2:])

	// Meta.
	case "version":
		fmt.Println(version)
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
  server setup [host]  Provision a server (local or remote via SSH)
  server doctor        Check server health via gRPC
  serve                Start the Outpost gRPC daemon

Client commands:
  login <host> <token> Connect to an Outpost server
  doctor               Check client health
  handoff              Hand off work to the server
  status               Dashboard with active runs + history
  logs <id>            Stream or dump run log output
  pickup <id>          Download completed patch
  drop <id>            Drop a run
  convert <id> <mode>  Convert between interactive/headless

Meta:
  version              Print version`)
}
