// Package server provides the MCP server implementation.
package server

import (
	"github.com/jbeshir/alignment-research-feed/cmd/mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server is the MCP server for the Alignment Research Feed.
type Server struct {
	client    *client.Client
	mcpServer *server.MCPServer
}

// NewServer creates a new MCP server with the given API client.
func NewServer(apiClient *client.Client) *Server {
	s := &Server{
		client: apiClient,
	}

	s.mcpServer = server.NewMCPServer(
		"alignment-feed",
		"1.0.0",
		server.WithResourceCapabilities(true, false),
		server.WithLogging(),
	)

	s.registerTools()
	s.registerResources()

	return s
}

// Run starts the MCP server with stdio transport.
func (s *Server) Run() error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	// search_articles - Search alignment research articles
	s.mcpServer.AddTool(mcp.NewTool("search_articles",
		mcp.WithDescription(
			"Search alignment research articles by keyword, source, or date range. "+
				"Returns a list of matching articles sorted by publication date "+
				"(newest first by default)."),
		mcp.WithString("query",
			mcp.Description("Search query to match in article titles"),
		),
		mcp.WithString("sources",
			mcp.Description("Comma-separated list of sources to include (e.g., 'arxiv,lesswrong')"),
		),
		mcp.WithString("exclude_sources",
			mcp.Description("Comma-separated list of sources to exclude"),
		),
		mcp.WithString("published_after",
			mcp.Description("Only include articles published after this date (RFC3339 format, e.g., '2024-01-01T00:00:00Z')"),
		),
		mcp.WithString("published_before",
			mcp.Description("Only include articles published before this date (RFC3339 format)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of articles to return (default: 20, max: 200)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (1-indexed, default: 1)"),
		),
	), s.handleSearchArticles)

	// get_article - Get full article details
	s.mcpServer.AddTool(mcp.NewTool("get_article",
		mcp.WithDescription("Get full details of a specific article by its ID (hash_id)."),
		mcp.WithString("article_id",
			mcp.Required(),
			mcp.Description("The hash_id of the article to retrieve"),
		),
	), s.handleGetArticle)

	// semantic_search - Search by text similarity
	s.mcpServer.AddTool(mcp.NewTool("semantic_search",
		mcp.WithDescription(
			"Search for articles semantically similar to the given text. "+
				"Useful for finding related research by providing a snippet, abstract, or topic description."),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("The text to find semantically similar articles for (max 100KB)"),
			mcp.MaxLength(102400),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of articles to return (default: 10, max: 100)"),
		),
	), s.handleSemanticSearch)

	// get_similar_articles - Find similar articles
	s.mcpServer.AddTool(mcp.NewTool("get_similar_articles",
		mcp.WithDescription(
			"Find articles similar to a given article using vector similarity. "+
				"Useful for discovering related research."),
		mcp.WithString("article_id",
			mcp.Required(),
			mcp.Description("The hash_id of the article to find similar articles for"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of similar articles to return (default: 10)"),
		),
	), s.handleGetSimilarArticles)

	// get_recommendations - Personalized recommendations
	s.mcpServer.AddTool(mcp.NewTool("get_recommendations",
		mcp.WithDescription(
			"Get personalized article recommendations based on your rating history. "+
				"Requires authentication."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of recommendations to return (default: 10)"),
		),
	), s.handleGetRecommendations)

	// rate_article - Set thumbs up/down
	s.mcpServer.AddTool(mcp.NewTool("rate_article",
		mcp.WithDescription("Rate an article with thumbs up or thumbs down. This affects your personalized recommendations."),
		mcp.WithString("article_id",
			mcp.Required(),
			mcp.Description("The hash_id of the article to rate"),
		),
		mcp.WithString("rating",
			mcp.Required(),
			mcp.Description("Rating to apply: 'up' for thumbs up, 'down' for thumbs down, 'none' to clear rating"),
		),
	), s.handleRateArticle)

	// mark_read - Mark as read/unread
	s.mcpServer.AddTool(mcp.NewTool("mark_read",
		mcp.WithDescription("Mark an article as read or unread."),
		mcp.WithString("article_id",
			mcp.Required(),
			mcp.Description("The hash_id of the article"),
		),
		mcp.WithBoolean("read",
			mcp.Required(),
			mcp.Description("Whether to mark as read (true) or unread (false)"),
		),
	), s.handleMarkRead)

	// list_liked - User's liked articles
	s.mcpServer.AddTool(mcp.NewTool("list_liked",
		mcp.WithDescription("List articles you have marked as liked (thumbs up). Requires authentication."),
		mcp.WithNumber("page",
			mcp.Description("Page number (1-indexed, default: 1)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Number of articles per page (default: 50, max: 200)"),
		),
	), s.handleListLiked)

	// list_disliked - User's disliked articles
	s.mcpServer.AddTool(mcp.NewTool("list_disliked",
		mcp.WithDescription("List articles you have marked as disliked (thumbs down). Requires authentication."),
		mcp.WithNumber("page",
			mcp.Description("Page number (1-indexed, default: 1)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Number of articles per page (default: 50, max: 200)"),
		),
	), s.handleListDisliked)

	// list_unreviewed - Unreviewed articles
	s.mcpServer.AddTool(mcp.NewTool("list_unreviewed",
		mcp.WithDescription("List articles you haven't reviewed yet. Requires authentication."),
		mcp.WithNumber("page",
			mcp.Description("Page number (1-indexed, default: 1)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Number of articles per page (default: 50, max: 200)"),
		),
	), s.handleListUnreviewed)
}
