package mcp

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Serve starts the MCP server on stdio.
func Serve(version string) error {
	s := server.NewMCPServer("outpost", version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithInstructions("Outpost remote Claude Code session runner. Use these tools to hand off tasks, check status, view logs, and pick up completed work."),
	)

	registerTools(s)

	return server.ServeStdio(s,
		server.WithErrorLogger(log.New(os.Stderr, "outpost-mcp: ", 0)),
	)
}

func registerTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("outpost_handoff",
			mcp.WithDescription("Hand off the current Claude Code session to a remote Outpost server for asynchronous execution. Creates a tar.gz archive of git-tracked files, reads the session JSONL, and streams everything to the server. The remote server spawns a new Claude session to continue the work."),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Claude session UUID to resume on the remote server"),
			),
			mcp.WithString("mode",
				mcp.Description("Execution mode on the remote server"),
				mcp.Enum("interactive", "headless"),
				mcp.DefaultString("interactive"),
			),
			mcp.WithString("name",
				mcp.Description("Human-readable name for the run"),
			),
			mcp.WithString("branch",
				mcp.Description("Git branch name for the remote repo"),
			),
			mcp.WithNumber("max_turns",
				mcp.Description("Maximum Claude turns before auto-stop"),
				mcp.DefaultNumber(50),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
		handleHandoff,
	)

	s.AddTool(
		mcp.NewTool("outpost_status",
			mcp.WithDescription("Check the status of Outpost runs. With no run_id, returns a dashboard of all runs with counts. With a run_id, returns detailed status for that specific run including log tail."),
			mcp.WithString("run_id",
				mcp.Description("Specific run ID to check. Omit for dashboard of all runs."),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		handleStatus,
	)

	s.AddTool(
		mcp.NewTool("outpost_logs",
			mcp.WithDescription("View recent log output from an Outpost run. Returns a snapshot of the last N lines (not a live stream). Call again for updated logs."),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The run ID to fetch logs for"),
			),
			mcp.WithNumber("lines",
				mcp.Description("Number of recent lines to return"),
				mcp.DefaultNumber(50),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		handleLogs,
	)

	s.AddTool(
		mcp.NewTool("outpost_pickup",
			mcp.WithDescription("Download the completed patch and forked session from an Outpost run. Returns file paths to the downloaded artifacts. Does NOT apply the patch or commit -- use git apply on the returned patch_path, then run checks and commit."),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The run ID to pick up"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		handlePickup,
	)

	s.AddTool(
		mcp.NewTool("outpost_drop",
			mcp.WithDescription("Stop and discard an Outpost run. The remote Claude session is killed and all run data is removed from the server."),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The run ID to drop"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
		handleDrop,
	)

	s.AddTool(
		mcp.NewTool("outpost_convert",
			mcp.WithDescription("Convert a running Outpost session between interactive and headless modes. Interactive mode allows tmux attachment; headless mode runs autonomously."),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The run ID to convert"),
			),
			mcp.WithString("target_mode",
				mcp.Required(),
				mcp.Description("The mode to convert to"),
				mcp.Enum("interactive", "headless"),
			),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
		handleConvert,
	)

	s.AddTool(
		mcp.NewTool("outpost_watch",
			mcp.WithDescription("Poll an Outpost run until it reaches a terminal state (complete, failed, or dropped). Blocks for up to timeout_minutes, polling every 30 seconds. Returns the final status and log tail when done."),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The run ID to watch"),
			),
			mcp.WithNumber("timeout_minutes",
				mcp.Description("Maximum minutes to wait before returning"),
				mcp.DefaultNumber(30),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		handleWatch,
	)

	s.AddTool(
		mcp.NewTool("outpost_doctor",
			mcp.WithDescription("Run health checks on the Outpost client configuration and server connectivity. Returns diagnostic information including server version, capacity, and installed tools."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		handleDoctor,
	)
}
