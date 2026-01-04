package command

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type RecommendArticles struct {
	ThumbsUpLister   datasources.ThumbsUpArticleLister
	SimilarityLister datasources.SimilarArticleLister
	ArticleFetcher   datasources.ArticleFetcher
}

func (c *RecommendArticles) Execute(ctx context.Context, userID string, limit int) ([]domain.Article, error) {
	// Get the articles the user has thumbs_up'd
	thumbsUpIDs, err := c.ThumbsUpLister.ListThumbsUpArticleIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(thumbsUpIDs) == 0 {
		return nil, nil
	}

	// Find similar articles based on the thumbs_up'd articles
	similarArticles, err := c.SimilarityLister.ListSimilarArticles(ctx, thumbsUpIDs, limit)
	if err != nil {
		return nil, err
	}

	if len(similarArticles) == 0 {
		return nil, nil
	}

	// Get the full article details
	ids := make([]string, 0, len(similarArticles))
	for _, similar := range similarArticles {
		ids = append(ids, similar.HashID)
	}

	return c.ArticleFetcher.FetchArticlesByID(ctx, ids)
}
