package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/store"
)

func handleStatus(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer client.Close()

	ctx := context.Background()
	runID := req.GetString("run_id", "")

	if runID != "" {
		return statusDetail(ctx, client, runID)
	}
	return statusDashboard(ctx, client)
}

func statusDashboard(ctx context.Context, client *grpcclient.Client) (*mcp.CallToolResult, error) {
	runs, err := client.ListRuns(ctx)
	if err != nil {
		return mcp.NewToolResultError("Failed to list runs: " + humanizeGRPCError(err)), nil
	}

	var active, complete, failed, dropped int
	var summaries []map[string]any

	for _, r := range runs {
		switch r.Status {
		case store.StatusPending, store.StatusRunning:
			active++
			summaries = append(summaries, runSummary(r))
		case store.StatusComplete:
			complete++
			summaries = append(summaries, runSummary(r))
		case store.StatusFailed:
			failed++
			summaries = append(summaries, runSummary(r))
		case store.StatusDropped:
			dropped++
		}
	}

	if summaries == nil {
		summaries = []map[string]any{}
	}

	result := map[string]any{
		"server":   serverAddress(),
		"active":   active,
		"complete": complete,
		"failed":   failed,
		"dropped":  dropped,
		"runs":     summaries,
	}

	return mcp.NewToolResultJSON(result)
}

func statusDetail(ctx context.Context, client *grpcclient.Client, id string) (*mcp.CallToolResult, error) {
	r, err := client.GetRun(ctx, id)
	if err != nil {
		return mcp.NewToolResultError("Failed to get run: " + humanizeGRPCError(err)), nil
	}

	return mcp.NewToolResultJSON(runToMap(r))
}
