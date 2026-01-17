package datasources

import (
	"context"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	LatestArticleLister
	ThumbsUpArticleLister
	UnreviewedArticleLister
	LikedArticleLister
	DislikedArticleLister
	ArticleFetcher
	ArticleReadSetter
	ArticleThumbsUpSetter
	ArticleThumbsDownSetter
	UserVectorGetter
	UserVectorSyncer
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

type UnreviewedArticleLister interface {
	ListUnreviewedArticleIDs(ctx context.Context, userID string, page, pageSize int) ([]string, error)
}

type LikedArticleLister interface {
	ListLikedArticleIDs(ctx context.Context, userID string, page, pageSize int) ([]string, error)
}

type DislikedArticleLister interface {
	ListDislikedArticleIDs(ctx context.Context, userID string, page, pageSize int) ([]string, error)
}

type UserVectorGetter interface {
	GetUserVector(ctx context.Context, userID string) (vectorSum []float32, count int, err error)
}

// UserVectorSyncer handles transactional updates to user recommendation vectors.
type UserVectorSyncer interface {
	// AddArticleVectorToUser atomically checks if the article's vector has already been added,
	// and if not, adds it to the user's vector sum and marks it as added.
	// Returns true if the vector was added, false if it was already added.
	AddArticleVectorToUser(
		ctx context.Context, userID, articleHashID string, vector []float32,
	) (added bool, err error)

	// SubtractArticleVectorFromUser atomically checks if the article's vector was previously added,
	// and if so, subtracts it from the user's vector sum and marks it as removed.
	// If vector is nil, only clears the added flag without modifying the sum.
	// Returns true if the flag was cleared, false if it wasn't set.
	SubtractArticleVectorFromUser(
		ctx context.Context, userID, articleHashID string, vector []float32,
	) (removed bool, err error)
}
