package mcp

import (
	"context"
	"errors"
	"io"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
)

func handleLogs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError("run_id is required"), nil
	}

	lines := req.GetInt("lines", 50)
	if lines <= 0 {
		lines = 50
	}

	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer client.Close()

	ctx := context.Background()

	stream, err := client.TailLogs(ctx, runID, false)
	if err != nil {
		return mcp.NewToolResultError("Failed to fetch logs: " + humanizeGRPCError(err)), nil
	}

	var allLines []string
	for {
		entry, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return mcp.NewToolResultError("Log stream error: " + humanizeGRPCError(recvErr)), nil
		}
		allLines = append(allLines, entry.GetLine())
	}

	totalLines := len(allLines)
	start := 0
	if totalLines > lines {
		start = totalLines - lines
	}

	result := map[string]any{
		"run_id":      runID,
		"lines":       allLines[start:],
		"total_lines": totalLines,
		"truncated":   start > 0,
	}

	return mcp.NewToolResultJSON(result)
}
