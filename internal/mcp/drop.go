package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
)

func handleDrop(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError("run_id is required"), nil
	}

	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer client.Close()

	droppedID, err := client.DropRun(context.Background(), runID)
	if err != nil {
		return mcp.NewToolResultError("Failed to drop run: " + humanizeGRPCError(err)), nil
	}

	result := map[string]any{
		"run_id": droppedID,
		"status": "dropped",
	}

	return mcp.NewToolResultJSON(result)
}
