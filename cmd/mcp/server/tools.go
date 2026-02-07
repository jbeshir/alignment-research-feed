package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jbeshir/alignment-research-feed/cmd/mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleSearchArticles(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	filters, err := parseSearchFilters(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	articles, err := s.client.SearchArticles(ctx, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search articles: %v", err)), nil
	}

	return formatArticlesResult(articles)
}

func parseSearchFilters(args map[string]any) (client.SearchFilters, error) {
	filters := client.SearchFilters{
		Limit: 20, // Default limit
	}

	parseStringFilters(args, &filters)

	if err := parseDateFilters(args, &filters); err != nil {
		return filters, err
	}

	parsePaginationFilters(args, &filters)

	return filters, nil
}

func parseStringFilters(args map[string]any, filters *client.SearchFilters) {
	if query, ok := args["query"].(string); ok && query != "" {
		filters.Query = query
	}
	if sources, ok := args["sources"].(string); ok && sources != "" {
		filters.Sources = splitAndTrim(sources)
	}
	if excludeSources, ok := args["exclude_sources"].(string); ok && excludeSources != "" {
		filters.ExcludeSources = splitAndTrim(excludeSources)
	}
}

func parseDateFilters(args map[string]any, filters *client.SearchFilters) error {
	if after, ok := args["published_after"].(string); ok && after != "" {
		t, err := time.Parse(time.RFC3339, after)
		if err != nil {
			return fmt.Errorf("invalid published_after date format: %w", err)
		}
		filters.PublishedAfter = &t
	}
	if before, ok := args["published_before"].(string); ok && before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return fmt.Errorf("invalid published_before date format: %w", err)
		}
		filters.PublishedBefore = &t
	}
	return nil
}

func parsePaginationFilters(args map[string]any, filters *client.SearchFilters) {
	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		filters.Limit = min(int(limit), 200)
	}
	if page, ok := args["page"].(float64); ok && page > 0 {
		filters.Page = int(page)
	}
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func (s *Server) handleSemanticSearch(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	text, ok := args["text"].(string)
	if !ok || text == "" {
		return mcp.NewToolResultError("text is required"), nil
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = min(int(l), 100)
	}

	articles, err := s.client.SemanticSearch(ctx, text, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search articles: %v", err)), nil
	}

	return formatArticlesResult(articles)
}

func (s *Server) handleGetArticle(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	articleID, ok := args["article_id"].(string)
	if !ok || articleID == "" {
		return mcp.NewToolResultError("article_id is required"), nil
	}

	article, err := s.client.GetArticle(ctx, articleID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get article: %v", err)), nil
	}

	return formatArticleResult(article)
}

func (s *Server) handleGetSimilarArticles(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	articleID, ok := args["article_id"].(string)
	if !ok || articleID == "" {
		return mcp.NewToolResultError("article_id is required"), nil
	}

	limit := 10 // Default limit
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	articles, err := s.client.GetSimilarArticles(ctx, articleID, limit)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get similar articles: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	return formatArticlesResult(articles)
}

func (s *Server) handleGetRecommendations(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	limit := 10 // Default limit
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	articles, err := s.client.GetRecommendations(ctx, limit)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get recommendations: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	return formatArticlesResult(articles)
}

func (s *Server) handleRateArticle(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	articleID, ok := args["article_id"].(string)
	if !ok || articleID == "" {
		return mcp.NewToolResultError("article_id is required"), nil
	}

	rating, ok := args["rating"].(string)
	if !ok || rating == "" {
		errMsg := "rating is required (must be 'up', 'down', or 'none')"
		return mcp.NewToolResultError(errMsg), nil
	}

	var thumbsUp, thumbsDown bool
	switch strings.ToLower(rating) {
	case "up":
		thumbsUp = true
		thumbsDown = false
	case "down":
		thumbsUp = false
		thumbsDown = true
	case "none":
		thumbsUp = false
		thumbsDown = false
	default:
		return mcp.NewToolResultError("rating must be 'up', 'down', or 'none'"), nil
	}

	err := s.client.RateArticle(ctx, articleID, thumbsUp, thumbsDown)
	if err != nil {
		errMsg := fmt.Sprintf("failed to rate article: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	msg := fmt.Sprintf("Successfully rated article %s as '%s'", articleID, rating)
	return mcp.NewToolResultText(msg), nil
}

func (s *Server) handleMarkRead(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	articleID, ok := args["article_id"].(string)
	if !ok || articleID == "" {
		return mcp.NewToolResultError("article_id is required"), nil
	}

	read, ok := args["read"].(bool)
	if !ok {
		return mcp.NewToolResultError("read is required (true or false)"), nil
	}

	err := s.client.MarkRead(ctx, articleID, read)
	if err != nil {
		errMsg := fmt.Sprintf("failed to mark article as read: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	status := "read"
	if !read {
		status = "unread"
	}
	msg := fmt.Sprintf("Successfully marked article %s as %s", articleID, status)
	return mcp.NewToolResultText(msg), nil
}

func (s *Server) handleListLiked(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	page, pageSize := parsePagination(request.Params.Arguments)

	articles, err := s.client.ListLiked(ctx, page, pageSize)
	if err != nil {
		errMsg := fmt.Sprintf("failed to list liked articles: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	return formatArticlesResult(articles)
}

func (s *Server) handleListDisliked(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	page, pageSize := parsePagination(request.Params.Arguments)

	articles, err := s.client.ListDisliked(ctx, page, pageSize)
	if err != nil {
		errMsg := fmt.Sprintf("failed to list disliked articles: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	return formatArticlesResult(articles)
}

func (s *Server) handleListUnreviewed(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	page, pageSize := parsePagination(request.Params.Arguments)

	articles, err := s.client.ListUnreviewed(ctx, page, pageSize)
	if err != nil {
		errMsg := fmt.Sprintf("failed to list unreviewed articles: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	return formatArticlesResult(articles)
}

func parsePagination(args map[string]any) (page, pageSize int) {
	page = 1
	pageSize = 50

	if p, ok := args["page"].(float64); ok && p > 0 {
		page = int(p)
	}
	if ps, ok := args["page_size"].(float64); ok && ps > 0 {
		pageSize = min(int(ps), 200)
	}
	return page, pageSize
}

func formatArticlesResult(articles []client.Article) (*mcp.CallToolResult, error) {
	if len(articles) == 0 {
		return mcp.NewToolResultText("No articles found."), nil
	}

	data, err := json.MarshalIndent(articles, "", "  ")
	if err != nil {
		errMsg := fmt.Sprintf("failed to format articles: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	msg := fmt.Sprintf("Found %d article(s):\n\n%s", len(articles), string(data))
	return mcp.NewToolResultText(msg), nil
}

func formatArticleResult(article *client.Article) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(article, "", "  ")
	if err != nil {
		errMsg := fmt.Sprintf("failed to format article: %v", err)
		return mcp.NewToolResultError(errMsg), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
