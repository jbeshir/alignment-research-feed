package datasources

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	LatestArticleLister
	ThumbsUpArticleLister
	ArticleFetcher
	ArticleReadSetter
	ArticleThumbsUpSetter
	ArticleThumbsDownSetter
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

type ArticleThumbsUpSetter interface {
	SetArticleThumbsUp(
		ctx context.Context,
		hashID string,
		userID string,
		thumbsUp bool,
	) error
}

type ArticleThumbsDownSetter interface {
	SetArticleThumbsDown(
		ctx context.Context,
		hashID string,
		userID string,
		thumbsDown bool,
	) error
}

type LatestArticleLister interface {
	ListLatestArticleIDs(
		ctx context.Context,
		filters domain.ArticleFilters,
		options domain.ArticleListOptions,
	) ([]string, error)
}

type ThumbsUpArticleLister interface {
	ListThumbsUpArticleIDs(ctx context.Context, userID string) ([]string, error)
}
