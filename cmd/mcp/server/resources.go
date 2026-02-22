package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerResources() {
	// Register the article resource template
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"article://{hash_id}",
			"Individual article from the alignment research feed",
			mcp.WithTemplateDescription(
				"Fetch a specific article by its hash_id. Use this to get full "+
					"article details including title, authors, link, source, "+
					"publication date, and LLM-generated analysis (summary, "+
					"key points, implication, and category)."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		s.handleArticleResource,
	)
}

func (s *Server) handleArticleResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	// Extract hash_id from the URI (format: article://{hash_id})
	uri := request.Params.URI
	if !strings.HasPrefix(uri, "article://") {
		return nil, fmt.Errorf("invalid article URI format: %s", uri)
	}

	hashID := strings.TrimPrefix(uri, "article://")
	if hashID == "" {
		return nil, fmt.Errorf("missing hash_id in URI: %s", uri)
	}

	article, err := s.client.GetArticle(ctx, hashID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch article %s: %w", hashID, err)
	}

	data, err := json.MarshalIndent(article, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal article: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
