package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/store"
)

//nolint:gocritic // mcp-go ToolHandlerFunc requires CallToolRequest by value
func handleWatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError("run_id is required"), nil
	}

	timeoutMinutes := req.GetInt("timeout_minutes", 30)
	if timeoutMinutes <= 0 {
		timeoutMinutes = 30
	}

	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer func() { _ = client.Close() }()

	deadline := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	start := time.Now()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Check immediately before waiting.
	r, err := client.GetRun(ctx, runID)
	if err != nil {
		return mcp.NewToolResultError("Failed to get run: " + humanizeGRPCError(err)), nil
	}

	if isTerminal(r.Status) {
		return watchResult(r, time.Since(start), false), nil
	}

	for {
		select {
		case <-ctx.Done():
			return watchResult(r, time.Since(start), true), nil
		case <-ticker.C:
			if time.Now().After(deadline) {
				return watchResult(r, time.Since(start), true), nil
			}

			r, err = client.GetRun(ctx, runID)
			if err != nil {
				return mcp.NewToolResultError("Failed to get run: " + humanizeGRPCError(err)), nil
			}

			if isTerminal(r.Status) {
				return watchResult(r, time.Since(start), false), nil
			}
		}
	}
}

func isTerminal(s store.Status) bool {
	return s == store.StatusComplete || s == store.StatusFailed || s == store.StatusDropped
}

func watchResult(r *store.Run, waited time.Duration, timedOut bool) *mcp.CallToolResult {
	result := map[string]any{
		"run_id":        r.ID,
		"status":        string(r.Status),
		"patch_ready":   r.PatchReady,
		"session_ready": r.SessionReady,
		"waited":        formatWaited(waited),
		"timed_out":     timedOut,
	}
	if r.LogTail != "" {
		result["log_tail"] = r.LogTail
	}

	var msg string
	switch {
	case timedOut:
		msg = fmt.Sprintf("Watch timed out after %s. Run is still %s.", formatWaited(waited), r.Status)
	case r.Status == store.StatusComplete:
		msg = "Run completed. Pick up with outpost_pickup."
	case r.Status == store.StatusFailed:
		msg = "Run failed. Check logs with outpost_logs or drop with outpost_drop."
	case r.Status == store.StatusDropped:
		msg = "Run was dropped."
	}
	result["message"] = msg

	// Best-effort JSON; fall back to text on marshal error.
	res, err := mcp.NewToolResultJSON(result)
	if err != nil {
		return mcp.NewToolResultText(msg)
	}
	return res
}

func formatWaited(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
