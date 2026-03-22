package mcp

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/store"
)

//nolint:gocritic // mcp-go ToolHandlerFunc requires CallToolRequest by value
func handleHandoff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	mode := req.GetString("mode", "interactive")
	name := req.GetString("name", "")
	branch := req.GetString("branch", "")
	maxTurns := req.GetInt("max_turns", 50)

	// Validate mode.
	var runMode store.Mode
	switch mode {
	case "interactive":
		runMode = store.ModeInteractive
	case "headless":
		runMode = store.ModeHeadless
	default:
		return mcp.NewToolResultErrorf("invalid mode %q: must be interactive or headless", mode), nil
	}

	// Read and truncate session JSONL.
	sessionJSONL, err := readSessionJSONL(sessionID)
	if err != nil {
		return mcp.NewToolResultError("Failed to read session: " + err.Error()), nil
	}

	// Create archive of git-tracked files.
	archivePath, err := createArchive()
	if err != nil {
		return mcp.NewToolResultError("Failed to create archive: " + err.Error()), nil
	}
	defer func() { _ = os.Remove(archivePath) }()

	// Connect to server.
	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer func() { _ = client.Close() }()

	// Preflight capacity check.
	doc, err := client.ServerDoctor(ctx)
	if err != nil {
		return mcp.NewToolResultError("Preflight check failed: " + humanizeGRPCError(err)), nil
	}
	if doc.ActiveRuns >= doc.MaxRuns {
		return mcp.NewToolResultErrorf(
			"Server at capacity (%d/%d active runs). Drop a run with outpost_drop first.",
			doc.ActiveRuns, doc.MaxRuns,
		), nil
	}

	// Stream archive to server.
	result, err := client.Handoff(ctx, archivePath, &grpcclient.HandoffMeta{
		SessionID:    sessionID,
		SessionJSONL: sessionJSONL,
		Mode:         store.ModeToProto(runMode),
		Name:         name,
		Branch:       branch,
		MaxTurns:     int32(maxTurns),
	}, nil)
	if err != nil {
		return mcp.NewToolResultError("Handoff failed: " + humanizeGRPCError(err)), nil
	}

	status := store.StatusFromProto(result.Status)

	response := map[string]any{
		"id":     result.ID,
		"status": string(status),
		"mode":   mode,
	}
	if result.Attach != "" {
		response["attach"] = result.Attach
	}
	if result.AttachLocal != "" {
		response["attach_local"] = result.AttachLocal
	}
	response["message"] = "Session handed off successfully. Run ID: " + result.ID

	return mcp.NewToolResultJSON(response)
}
