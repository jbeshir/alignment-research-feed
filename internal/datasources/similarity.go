package datasources

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// SimilarityRepository combines all similarity-related interfaces.
type SimilarityRepository interface {
	SimilarArticleLister
	ArticleVectorFetcher
	SimilarArticlesByVectorLister
}

type SimilarArticleLister interface {
	ListSimilarArticles(
		ctx context.Context,
		hashIDs []string,
		count int,
	) ([]domain.SimilarArticle, error)
}

type ArticleVectorFetcher interface {
	FetchArticleVector(ctx context.Context, hashID string) ([]float32, error)
}

type SimilarArticlesByVectorLister interface {
	ListSimilarArticlesByVector(
		ctx context.Context,
		excludeHashIDs []string,
		vector []float32,
		limit int,
	) ([]domain.SimilarArticle, error)
}

// NullSimilarityRepository is a null implementation of SimilarityRepository.
type NullSimilarityRepository struct{}

var _ SimilarityRepository = NullSimilarityRepository{}

func (NullSimilarityRepository) ListSimilarArticles(
	_ context.Context,
	_ []string,
	_ int,
) ([]domain.SimilarArticle, error) {
	return nil, nil
}

func (NullSimilarityRepository) FetchArticleVector(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

func (NullSimilarityRepository) ListSimilarArticlesByVector(
	_ context.Context,
	_ []string,
	_ []float32,
	_ int,
) ([]domain.SimilarArticle, error) {
	return nil, nil
}
