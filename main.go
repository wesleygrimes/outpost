package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/wesleygrimes/outpost/cmd"
	"github.com/wesleygrimes/outpost/internal/ui"
)

// version is set by -ldflags at build time.
var version = "dev"

func main() {
	cmd.Version = version
	ui.Init(version)

	args := extractGlobalFlags(os.Args[1:])

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	subcmd := args[0]
	subargs := args[1:]
	var err error

	switch subcmd {
	// Server commands.
	case "server":
		err = cmd.Server(subargs)
	case "serve":
		err = cmd.Serve()

	// Client commands.
	case "login":
		err = cmd.Login(subargs)
	case "doctor":
		err = cmd.Doctor()
	case "handoff":
		err = cmd.Handoff(subargs)
	case "status":
		err = cmd.Status(subargs)
	case "logs":
		err = cmd.Logs(subargs)
	case "pickup":
		err = cmd.Pickup(subargs)
	case "drop":
		err = cmd.Drop(subargs)
	case "convert":
		err = cmd.Convert(subargs)
	case "mcp":
		err = cmd.MCP()

	// Meta.
	case "version":
		fmt.Println(version)
	default:
		ui.Errf("error: unknown command %q\n", subcmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		var displayed *cmd.DisplayedError
		if !errors.As(err, &displayed) {
			ui.Errf("error: %v\n", err)
		}
		os.Exit(1)
	}
}

// extractGlobalFlags strips --no-color, --quiet, and --force from args
// and applies them to the ui package. Remaining args are returned.
func extractGlobalFlags(args []string) []string {
	remaining := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--no-color":
			ui.SetColor(false)
		case "--quiet", "-q":
			ui.QuietMode = true
		case "--force":
			ui.ForceMode = true
		default:
			remaining = append(remaining, arg)
		}
	}
	return remaining
}

func printUsage() {
	ui.Errln(`Usage: outpost <command> [flags]

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
  mcp                  Start MCP server (stdio transport)

Meta:
  version              Print version

Global flags:
  --no-color           Disable color output
  --quiet, -q          Suppress non-essential output
  --force              Skip confirmation prompts`)
}
