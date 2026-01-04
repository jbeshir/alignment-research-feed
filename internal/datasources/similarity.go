package datasources

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type SimilarArticleLister interface {
	ListSimilarArticles(
		ctx context.Context,
		hashIDs []string,
		count int,
	) ([]domain.SimilarArticle, error)
}

// SimilarityRepository is an alias for SimilarArticleLister for semantic clarity.
type SimilarityRepository = SimilarArticleLister

type NullSimilarArticleLister struct{}

func (NullSimilarArticleLister) ListSimilarArticles(
	ctx context.Context,
	hashIDs []string,
	count int,
) ([]domain.SimilarArticle, error) {
	return nil, nil
}
