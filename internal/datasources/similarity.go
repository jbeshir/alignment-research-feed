package datasources

import (
	"context"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type SimilarityRepository interface {
	SimilarArticleLister
}

type SimilarArticleLister interface {
	ListSimilarArticles(
		ctx context.Context,
		id string,
		count int,
	) ([]domain.SimilarArticle, error)
}

type NullSimilarityRepository struct{}

func (NullSimilarityRepository) ListSimilarArticles(
	ctx context.Context,
	id string,
	count int,
) ([]domain.SimilarArticle, error) {
	return nil, nil
}
