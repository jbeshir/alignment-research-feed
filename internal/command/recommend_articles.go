package command

import (
	"context"
	"fmt"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// RecommendArticlesRequest is the request for the RecommendArticles command.
type RecommendArticlesRequest struct {
	UserID string
	Limit  int
}

// RecommendArticlesConfig holds configuration for serving recommendations.
type RecommendArticlesConfig struct {
	// PrecomputedStaleThreshold is the age after which precomputed recommendations
	// are considered stale and on-demand generation is used instead.
	PrecomputedStaleThreshold time.Duration

	// PrecomputedFetchLimit is how many precomputed recommendations to fetch.
	// This should be higher than the return limit to account for read article filtering.
	PrecomputedFetchLimit int
}

// RecommendArticles serves personalized article recommendations.
// It uses precomputed recommendations when available and fresh,
// falling back to on-demand generation via GenerateRecommendations.
type RecommendArticles struct {
	GenerateCommand    *GenerateRecommendations
	PrecomputedReader  datasources.PrecomputedRecommendationReader
	ReadArticlesLister datasources.ReadArticleIDsLister
	ArticleFetcher     datasources.ArticleFetcher
	Config             RecommendArticlesConfig
}

// NewRecommendArticles creates a properly initialized RecommendArticles command.
func NewRecommendArticles(
	generateCommand *GenerateRecommendations,
	precomputedReader datasources.PrecomputedRecommendationReader,
	readArticlesLister datasources.ReadArticleIDsLister,
	articleFetcher datasources.ArticleFetcher,
	config RecommendArticlesConfig,
) *RecommendArticles {
	return &RecommendArticles{
		GenerateCommand:    generateCommand,
		PrecomputedReader:  precomputedReader,
		ReadArticlesLister: readArticlesLister,
		ArticleFetcher:     articleFetcher,
		Config:             config,
	}
}

// Execute returns recommendations for a user.
// It first tries to use precomputed recommendations if available and fresh,
// then falls back to on-demand generation. Finally, it fetches full article data.
func (c *RecommendArticles) Execute(ctx context.Context, req RecommendArticlesRequest) ([]domain.Article, error) {
	logger := domain.LoggerFromContext(ctx)

	// Try precomputed recommendations first
	scored, err := c.getPrecomputedRecommendations(ctx, req.UserID, req.Limit)
	if err != nil {
		logger.WarnContext(ctx, "failed to get precomputed recommendations, falling back to on-demand",
			"error", err)
	}

	// Fall back to on-demand generation if precomputed is empty
	if len(scored) == 0 {
		scored, err = c.GenerateCommand.Execute(ctx, GenerateRecommendationsRequest(req))
		if err != nil {
			return nil, err
		}
	}

	if len(scored) == 0 {
		return nil, nil
	}

	// Fetch full article data
	ids := make([]string, len(scored))
	for i, s := range scored {
		ids[i] = s.HashID
	}

	articles, err := c.ArticleFetcher.FetchArticlesByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetching article details: %w", err)
	}

	return articles, nil
}

// getPrecomputedRecommendations retrieves and filters precomputed recommendations.
func (c *RecommendArticles) getPrecomputedRecommendations(
	ctx context.Context, userID string, limit int,
) ([]ScoredArticle, error) {
	logger := domain.LoggerFromContext(ctx)

	// Check if precomputed recommendations exist and are fresh
	generatedAt, err := c.PrecomputedReader.GetPrecomputedRecommendationAge(ctx, userID)
	if err != nil {
		return nil, err
	}

	if generatedAt.IsZero() {
		return nil, nil
	}

	if time.Since(generatedAt) > c.Config.PrecomputedStaleThreshold {
		logger.DebugContext(ctx, "precomputed recommendations are stale",
			"generated_at", generatedAt, "age", time.Since(generatedAt))
		return nil, nil
	}

	// Fetch precomputed recommendations
	precomputed, err := c.PrecomputedReader.GetPrecomputedRecommendations(ctx, userID, c.Config.PrecomputedFetchLimit)
	if err != nil {
		return nil, err
	}

	if len(precomputed) == 0 {
		return nil, nil
	}

	// Get read article IDs to filter
	readIDs := c.getReadArticleIDs(ctx, userID)

	// Filter out read articles and convert to ScoredArticle (maintaining order by position)
	result := make([]ScoredArticle, 0, limit)
	for _, rec := range precomputed {
		if _, isRead := readIDs[rec.ArticleHashID]; isRead {
			continue
		}
		result = append(result, ScoredArticle{
			HashID: rec.ArticleHashID,
			Score:  rec.Score,
			Source: rec.Source,
		})
		if len(result) >= limit {
			break
		}
	}

	return result, nil
}

// getReadArticleIDs fetches the set of article IDs the user has already read.
func (c *RecommendArticles) getReadArticleIDs(ctx context.Context, userID string) map[string]struct{} {
	logger := domain.LoggerFromContext(ctx)
	readIDs, err := c.ReadArticlesLister.ListReadArticleIDs(ctx, userID)
	if err != nil {
		logger.WarnContext(ctx, "failed to get read article IDs", "error", err)
		return make(map[string]struct{})
	}

	result := make(map[string]struct{}, len(readIDs))
	for _, id := range readIDs {
		result[id] = struct{}{}
	}
	return result
}
