package datasources

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type SimilarArticleLister interface {
	ListSimilarArticles(
		ctx context.Context,
		id string,
		count int,
	) ([]domain.SimilarArticle, error)
}

type NullSimilarArticleLister struct{}

func (NullSimilarArticleLister) ListSimilarArticles(
	ctx context.Context,
	id string,
	count int,
) ([]domain.SimilarArticle, error) {
	return nil, nil
}
