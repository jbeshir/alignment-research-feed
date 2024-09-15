package datasources

import (
	"context"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	LatestArticleLister
	ArticleFetcher
}

type ArticleFetcher interface {
	FetchArticlesByID(
		ctx context.Context,
		hashIDs []string,
	) ([]domain.Article, error)
}

type LatestArticleLister interface {
	ListLatestArticles(
		ctx context.Context,
		filters domain.ArticleFilters,
		options domain.ArticleListOptions,
	) ([]domain.Article, error)
}
