package cmd

import outpostmcp "github.com/wesleygrimes/outpost/internal/mcp"

// MCP starts the MCP server on stdio.
func MCP() error {
	return outpostmcp.Serve(Version)
}
