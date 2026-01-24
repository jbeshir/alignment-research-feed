package command

import (
	"context"
	"fmt"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// SetArticleRatingRequest is the request for the SetArticleRating command.
type SetArticleRatingRequest struct {
	UserID        string
	ArticleHashID string
	ThumbsUp      bool
	ThumbsDown    bool
}

// SetArticleRating handles setting article ratings (thumbs up/down) with vector sync.
// It fetches the article vector from Pinecone, then atomically updates the rating
// and syncs the aggregate user vector.
type SetArticleRating struct {
	ArticleVectorFetcher datasources.ArticleVectorFetcher
	RatingSetter         datasources.ArticleRatingSetter
	RegenerationMarker   datasources.UserRegenerationNeededMarker
}

// NewSetArticleRating creates a properly initialized SetArticleRating command.
func NewSetArticleRating(
	articleVectorFetcher datasources.ArticleVectorFetcher,
	ratingSetter datasources.ArticleRatingSetter,
	regenerationMarker datasources.UserRegenerationNeededMarker,
) *SetArticleRating {
	return &SetArticleRating{
		ArticleVectorFetcher: articleVectorFetcher,
		RatingSetter:         ratingSetter,
		RegenerationMarker:   regenerationMarker,
	}
}

// Execute sets the article rating and handles aggregate vector sync.
func (c *SetArticleRating) Execute(ctx context.Context, req SetArticleRatingRequest) (Empty, error) {
	logger := domain.LoggerFromContext(ctx)

	// 1. Fetch vector from Pinecone (graceful failure - nil is OK)
	vector, err := c.ArticleVectorFetcher.FetchArticleVector(ctx, req.ArticleHashID)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch article vector, proceeding without vector",
			"error", err, "articleHashID", req.ArticleHashID)
		vector = nil
	}
	if vector == nil {
		logger.DebugContext(ctx, "article has no vector", "articleHashID", req.ArticleHashID)
	}

	// 2. Set rating atomically (handles aggregate sync internally)
	if err := c.RatingSetter.SetArticleRating(ctx, req.UserID, req.ArticleHashID,
		req.ThumbsUp, req.ThumbsDown, vector); err != nil {
		return Empty{}, fmt.Errorf("setting article rating: %w", err)
	}

	logger.DebugContext(ctx, "set article rating",
		"articleHashID", req.ArticleHashID, "thumbsUp", req.ThumbsUp, "thumbsDown", req.ThumbsDown)

	// 3. Mark for regeneration (best-effort)
	if err := c.RegenerationMarker.MarkUserNeedsRegeneration(ctx, req.UserID); err != nil {
		logger.WarnContext(ctx, "failed to mark user for regeneration", "error", err)
	}

	return Empty{}, nil
}
