// Package main provides the entry point for the Alignment Research Feed MCP server.
//
// This MCP server allows AI agents (Claude Code, Cursor, Cline, Windsurf) to interact
// with the alignment research feed programmatically.
//
// Configuration:
//
//	ALIGNMENT_FEED_API_URL   - Base URL of the API (default: https://api.alignmentfeed.org)
//	ALIGNMENT_FEED_API_TOKEN - API token for authentication (required, format: user_api|xxx)
//
// Usage with Claude Code:
//
//	claude mcp add alignment-feed --transport stdio \
//	  --env ALIGNMENT_FEED_API_TOKEN=user_api|xxx \
//	  -- /path/to/alignment-feed-mcp
package main

import (
	"log"
	"os"

	"github.com/jbeshir/alignment-research-feed/cmd/mcp/client"
	"github.com/jbeshir/alignment-research-feed/cmd/mcp/server"
)

func main() {
	apiURL := os.Getenv("ALIGNMENT_FEED_API_URL")
	if apiURL == "" {
		apiURL = "https://api.alignmentfeed.org"
	}

	apiToken := os.Getenv("ALIGNMENT_FEED_API_TOKEN")
	if apiToken == "" {
		log.Fatal("ALIGNMENT_FEED_API_TOKEN environment variable is required")
	}

	apiClient := client.NewClient(apiURL, apiToken)
	srv := server.NewServer(apiClient)

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
