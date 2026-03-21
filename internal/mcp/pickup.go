package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
)

func handlePickup(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError("run_id is required"), nil
	}

	client, err := grpcclient.Load()
	if err != nil {
		return mcp.NewToolResultError("Outpost not configured. Run 'outpost login <host> <token>' first."), nil
	}
	defer client.Close()

	ctx := context.Background()

	r, err := client.GetRun(ctx, runID)
	if err != nil {
		return mcp.NewToolResultError("Failed to get run: " + humanizeGRPCError(err)), nil
	}
	if !r.PatchReady {
		return mcp.NewToolResultErrorf("No patch available for run %s (status=%s)", runID, r.Status), nil
	}

	// Download patch.
	patchDir := ".outpost/patches"
	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		return mcp.NewToolResultError("Failed to create patches directory: " + err.Error()), nil
	}

	patchPath := filepath.Join(patchDir, runID+".patch")
	if err := client.DownloadPatch(ctx, runID, patchPath); err != nil {
		return mcp.NewToolResultError("Failed to download patch: " + humanizeGRPCError(err)), nil
	}

	// Download forked session if available.
	var sessionID string
	if r.SessionReady && r.ForkedSessionID != "" {
		destPath, err := downloadSession(runID, r.ForkedSessionID)
		if err == nil {
			if dlErr := client.DownloadSession(ctx, runID, destPath); dlErr == nil {
				sessionID = r.ForkedSessionID
			}
		}
	}

	// Cleanup server-side run data.
	_ = client.CleanupRun(ctx, runID)

	// Get diff stat for context.
	diffStat := getDiffStat(patchPath)

	result := map[string]any{
		"run_id":     runID,
		"patch_path": patchPath,
	}
	if sessionID != "" {
		result["session_id"] = sessionID
	}
	if diffStat != "" {
		result["diff_stat"] = diffStat
	}
	result["message"] = fmt.Sprintf("Patch downloaded. Apply with: git apply %s", patchPath)

	return mcp.NewToolResultJSON(result)
}

func getDiffStat(patchPath string) string {
	cmd := exec.Command("git", "apply", "--stat", patchPath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
