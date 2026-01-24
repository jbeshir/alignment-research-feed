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
// On-demand results are stored for subsequent requests.
type RecommendArticles struct {
	GenerateCommand    *GenerateRecommendations
	PrecomputedReader  datasources.PrecomputedRecommendationReader
	PrecomputedWriter  datasources.PrecomputedRecommendationWriter
	RegenerationStatus datasources.UserRegeneratedMarker
	ReadArticlesLister datasources.ReadArticleIDsLister
	ArticleFetcher     datasources.ArticleFetcher
	Config             RecommendArticlesConfig
}

// NewRecommendArticles creates a properly initialized RecommendArticles command.
func NewRecommendArticles(
	generateCommand *GenerateRecommendations,
	precomputedReader datasources.PrecomputedRecommendationReader,
	precomputedWriter datasources.PrecomputedRecommendationWriter,
	regenerationStatus datasources.UserRegeneratedMarker,
	readArticlesLister datasources.ReadArticleIDsLister,
	articleFetcher datasources.ArticleFetcher,
	config RecommendArticlesConfig,
) *RecommendArticles {
	return &RecommendArticles{
		GenerateCommand:    generateCommand,
		PrecomputedReader:  precomputedReader,
		PrecomputedWriter:  precomputedWriter,
		RegenerationStatus: regenerationStatus,
		ReadArticlesLister: readArticlesLister,
		ArticleFetcher:     articleFetcher,
		Config:             config,
	}
}

// Execute returns recommendations for a user.
// It first tries to use precomputed recommendations if available and fresh,
// then falls back to on-demand generation and stores the results.
// Finally, it fetches full article data.
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

		// Store generated recommendations for next request
		if len(scored) > 0 {
			c.storeGeneratedRecommendations(ctx, req.UserID, scored)
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

// storeGeneratedRecommendations stores on-demand generated recommendations
// so they're available from the precomputed store on subsequent requests.
// Errors are logged but not returned since this is best-effort caching.
func (c *RecommendArticles) storeGeneratedRecommendations(
	ctx context.Context, userID string, scored []ScoredArticle,
) {
	logger := domain.LoggerFromContext(ctx)

	// Delete existing precomputed recommendations for this user
	if err := c.PrecomputedWriter.DeleteUserPrecomputedRecommendations(ctx, userID); err != nil {
		logger.WarnContext(ctx, "failed to delete existing precomputed recommendations",
			"user_id", userID, "error", err)
		return
	}

	// Store the new recommendations
	generatedAt := time.Now()
	for position, article := range scored {
		if err := c.PrecomputedWriter.UpsertPrecomputedRecommendation(
			ctx,
			userID,
			article.HashID,
			article.Score,
			article.Source,
			position,
			generatedAt,
		); err != nil {
			logger.WarnContext(ctx, "failed to store precomputed recommendation",
				"user_id", userID, "position", position, "error", err)
			return
		}
	}

	// Mark user as regenerated so background job doesn't redo work
	if err := c.RegenerationStatus.MarkUserRegenerated(ctx, userID); err != nil {
		logger.WarnContext(ctx, "failed to mark user as regenerated",
			"user_id", userID, "error", err)
	}

	logger.DebugContext(ctx, "stored on-demand recommendations",
		"user_id", userID, "count", len(scored))
}
