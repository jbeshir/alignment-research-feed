package datasources

import (
	"context"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	ArticleLister
}

type ArticleLister interface {
	ListLatestArticles(ctx context.Context, filters domain.ArticleFilters, limit int) ([]domain.Article, error)
	TotalMatchingArticles(ctx context.Context, filters domain.ArticleFilters) (int64, error)
}
