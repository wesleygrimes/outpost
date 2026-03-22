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

	if err := run(args[0], args[1:]); err != nil {
		var displayed *cmd.DisplayedError
		if !errors.As(err, &displayed) {
			ui.Errf("error: %v\n", err)
		}
		os.Exit(1)
	}
}

func run(subcmd string, subargs []string) error {
	switch subcmd {
	// Server commands.
	case "server":
		return cmd.Server(subargs)
	case "serve":
		return cmd.Serve()

	// Client commands.
	case "login":
		return cmd.Login(subargs)
	case "doctor":
		return cmd.Doctor()
	case "handoff":
		return cmd.Handoff(subargs)
	case "status":
		return cmd.Status(subargs)
	case "logs":
		return cmd.Logs(subargs)
	case "pickup":
		return cmd.Pickup(subargs)
	case "drop":
		return cmd.Drop(subargs)
	case "convert":
		return cmd.Convert(subargs)
	case "mcp":
		return cmd.MCP()

	// Meta.
	case "version":
		fmt.Println(version)
		return nil
	default:
		ui.Errf("error: unknown command %q\n", subcmd)
		printUsage()
		os.Exit(1)
		return nil
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
