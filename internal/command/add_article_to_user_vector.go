package command

import (
	"context"
	"fmt"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// AddArticleToUserVector handles adding a single article's vector to the user's recommendation vector.
// It should be called after a thumbs up is set to true.
type AddArticleToUserVector struct {
	ArticleVectorFetcher datasources.ArticleVectorFetcher
	UserVectorSyncer     datasources.UserVectorSyncer
}

// Execute adds the article's vector to the user's recommendation vector if not already added.
func (c *AddArticleToUserVector) Execute(ctx context.Context, userID, articleHashID string) error {
	logger := domain.LoggerFromContext(ctx)

	vector, err := c.ArticleVectorFetcher.FetchArticleVector(ctx, articleHashID)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch article vector, skipping sync",
			"error", err, "articleHashID", articleHashID)
		return nil
	}
	if vector == nil {
		logger.DebugContext(ctx, "article has no vector, skipping sync", "articleHashID", articleHashID)
		return nil
	}

	added, err := c.UserVectorSyncer.AddArticleVectorToUser(ctx, userID, articleHashID, vector)
	if err != nil {
		return fmt.Errorf("adding article vector to user: %w", err)
	}
	if added {
		logger.DebugContext(ctx, "added article vector to user", "articleHashID", articleHashID)
	}
	return nil
}
