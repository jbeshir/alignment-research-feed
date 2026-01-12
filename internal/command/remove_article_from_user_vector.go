package command

import (
	"context"
	"fmt"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// RemoveArticleFromUserVector handles removing a single article's vector from the user's recommendation vector.
// It should be called after a thumbs up is set to false or thumbs down is set to true.
type RemoveArticleFromUserVector struct {
	ArticleVectorFetcher datasources.ArticleVectorFetcher
	UserVectorSyncer     datasources.UserVectorSyncer
}

// Execute removes the article's vector from the user's recommendation vector if it was previously added.
func (c *RemoveArticleFromUserVector) Execute(ctx context.Context, userID, articleHashID string) error {
	logger := domain.LoggerFromContext(ctx)

	// Fetch vector (may be nil if article was deleted from Pinecone)
	vector, err := c.ArticleVectorFetcher.FetchArticleVector(ctx, articleHashID)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch article vector, skipping sync",
			"error", err, "articleHashID", articleHashID)
		return nil
	}

	removed, err := c.UserVectorSyncer.SubtractArticleVectorFromUser(ctx, userID, articleHashID, vector)
	if err != nil {
		return fmt.Errorf("subtracting article vector from user: %w", err)
	}
	if removed {
		logger.DebugContext(ctx, "subtracted article vector from user", "articleHashID", articleHashID)
	}
	return nil
}
