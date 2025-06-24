package datasources

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	LatestArticleLister
	ArticleFetcher
	ArticleReadSetter
}

type ArticleFetcher interface {
	FetchArticlesByID(
		ctx context.Context,
		hashIDs []string,
	) ([]domain.Article, error)
}

type ArticleReadSetter interface {
	SetArticleRead(
		ctx context.Context,
		hashID string,
		userID string,
		read bool,
	) error
}

type LatestArticleLister interface {
	ListLatestArticleIDs(
		ctx context.Context,
		filters domain.ArticleFilters,
		options domain.ArticleListOptions,
	) ([]string, error)
}
