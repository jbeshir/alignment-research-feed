package command

import (
	"context"
	"fmt"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type RecommendArticles struct {
	VectorSimilarity datasources.SimilarArticlesByVectorLister
	ArticleFetcher   datasources.ArticleFetcher
	UserVectorGetter datasources.UserVectorGetter
}

func (c *RecommendArticles) Execute(ctx context.Context, userID string, limit int) ([]domain.Article, error) {
	// Get stored vector (sum / count = average)
	vectorSum, count, err := c.UserVectorGetter.GetUserVector(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user vector: %w", err)
	}

	if vectorSum == nil || count == 0 {
		return nil, nil // No preferences yet
	}

	// Calculate average vector from sum
	avgVector := divideVector(vectorSum, float32(count))

	// Query Pinecone with the average vector
	// Note: We allow previously seen articles to be recommended for now
	similarArticles, err := c.VectorSimilarity.ListSimilarArticlesByVector(ctx, nil, avgVector, limit)
	if err != nil {
		return nil, fmt.Errorf("finding similar articles: %w", err)
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

func divideVector(vector []float32, divisor float32) []float32 {
	result := make([]float32, len(vector))
	for i, v := range vector {
		result[i] = v / divisor
	}
	return result
}
