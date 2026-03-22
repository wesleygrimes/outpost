package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/wesleygrimes/outpost/internal/grpcclient"
)

//nolint:gocritic // mcp-go ToolHandlerFunc requires CallToolRequest by value
func handleDoctor(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer func() { _ = client.Close() }()

	doc, err := client.ServerDoctor(ctx)
	if err != nil {
		return mcp.NewToolResultError("Server unreachable: " + humanizeGRPCError(err)), nil
	}

	result := map[string]any{
		"healthy":          true,
		"server":           serverAddress(),
		"server_version":   doc.Version,
		"uptime":           doc.Uptime,
		"disk_free":        doc.DiskFree,
		"claude_installed": doc.ClaudeInstalled,
		"tmux_installed":   doc.TmuxInstalled,
		"active_runs":      doc.ActiveRuns,
		"max_runs":         doc.MaxRuns,
		"total_runs":       doc.TotalRuns,
	}

	return mcp.NewToolResultJSON(result)
}
