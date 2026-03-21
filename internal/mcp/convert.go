package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/store"
)

func handleConvert(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError("run_id is required"), nil
	}

	targetStr, err := req.RequireString("target_mode")
	if err != nil {
		return mcp.NewToolResultError("target_mode is required"), nil
	}

	var targetMode store.Mode
	switch targetStr {
	case "interactive":
		targetMode = store.ModeInteractive
	case "headless":
		targetMode = store.ModeHeadless
	default:
		return mcp.NewToolResultErrorf("invalid mode %q: must be interactive or headless", targetStr), nil
	}

	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer client.Close()

	r, err := client.ConvertMode(context.Background(), runID, store.ModeToProto(targetMode))
	if err != nil {
		return mcp.NewToolResultError("Failed to convert mode: " + humanizeGRPCError(err)), nil
	}

	result := map[string]any{
		"run_id": r.ID,
		"mode":   string(r.Mode),
		"status": string(r.Status),
	}
	if r.Attach != "" {
		result["attach"] = r.Attach
	}
	if r.AttachLocal != "" {
		result["attach_local"] = r.AttachLocal
	}

	return mcp.NewToolResultJSON(result)
}
