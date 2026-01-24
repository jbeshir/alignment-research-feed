package datasources

import (
	"context"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	LatestArticleLister
	ThumbsUpArticleLister
	UnreviewedArticleLister
	LikedArticleLister
	DislikedArticleLister
	ReadArticleIDsLister
	ArticleFetcher
	ArticleReadSetter
	UserArticleInteractionStore
	UserInterestClusterStore
	PrecomputedRecommendationStore
	UserRecommendationStateStore
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

// ReadArticleIDsLister lists all article IDs a user has marked as read.
type ReadArticleIDsLister interface {
	ListReadArticleIDs(ctx context.Context, userID string) ([]string, error)
}

// ArticleRatingSetter atomically sets thumbs up/down.
type ArticleRatingSetter interface {
	SetArticleRating(
		ctx context.Context, userID, articleHashID string,
		thumbsUp, thumbsDown bool, vector []float32,
	) error
}

// UserArticleVectorsGetter retrieves article vectors for a user filtered by rating type.
type UserArticleVectorsGetter interface {
	GetUserArticleVectorsByType(
		ctx context.Context, userID string, ratingType domain.UserRatingType,
	) ([]domain.UserArticleRating, error)
}

// UserArticleVectorsCounter returns the count of vectors for a user by rating type.
type UserArticleVectorsCounter interface {
	CountUserArticleVectorsByType(
		ctx context.Context, userID string, ratingType domain.UserRatingType,
	) (int64, error)
}

// UserArticleInteractionStore combines all user-article interaction operations.
type UserArticleInteractionStore interface {
	ArticleRatingSetter
	UserArticleVectorsGetter
	UserArticleVectorsCounter
}

// UserInterestCluster represents a cluster centroid for a user's interests.
type UserInterestCluster struct {
	ClusterID      int
	CentroidVector []float32
	ArticleCount   int
	UpdatedAt      time.Time
}

// UserInterestClusterUpserter stores or updates a user's interest cluster.
type UserInterestClusterUpserter interface {
	UpsertUserInterestCluster(
		ctx context.Context, userID string, clusterID int, centroidVector []float32, articleCount int,
	) error
}

// UserInterestClusterGetter retrieves all interest clusters for a user.
type UserInterestClusterGetter interface {
	GetUserInterestClusters(ctx context.Context, userID string) ([]UserInterestCluster, error)
}

// UserInterestClusterDeleter removes all interest clusters for a user.
type UserInterestClusterDeleter interface {
	DeleteUserInterestClusters(ctx context.Context, userID string) error
}

// UserInterestClusterWriter combines cluster write operations.
type UserInterestClusterWriter interface {
	UserInterestClusterUpserter
	UserInterestClusterDeleter
}

// UserInterestClusterStore combines all user interest cluster operations.
type UserInterestClusterStore interface {
	UserInterestClusterWriter
	UserInterestClusterGetter
}

// PrecomputedRecommendation represents a stored recommendation for a user.
type PrecomputedRecommendation struct {
	ArticleHashID string
	Score         float64
	Source        string
	Position      int
	GeneratedAt   time.Time
}

// PrecomputedRecommendationUpserter stores or updates a precomputed recommendation.
type PrecomputedRecommendationUpserter interface {
	UpsertPrecomputedRecommendation(
		ctx context.Context,
		userID, articleHashID string,
		score float64,
		source string,
		position int,
		generatedAt time.Time,
	) error
}

// PrecomputedRecommendationDeleter removes all precomputed recommendations for a user.
type PrecomputedRecommendationDeleter interface {
	DeleteUserPrecomputedRecommendations(ctx context.Context, userID string) error
}

// PrecomputedRecommendationGetter retrieves precomputed recommendations for a user, ordered by rank.
type PrecomputedRecommendationGetter interface {
	GetPrecomputedRecommendations(ctx context.Context, userID string, limit int) ([]PrecomputedRecommendation, error)
}

// PrecomputedRecommendationAgeGetter returns when recommendations were last generated for a user.
type PrecomputedRecommendationAgeGetter interface {
	GetPrecomputedRecommendationAge(ctx context.Context, userID string) (time.Time, error)
}

// PrecomputedRecommendationReader combines read operations for precomputed recommendations.
type PrecomputedRecommendationReader interface {
	PrecomputedRecommendationGetter
	PrecomputedRecommendationAgeGetter
}

// PrecomputedRecommendationWriter combines write operations for precomputed recommendations.
type PrecomputedRecommendationWriter interface {
	PrecomputedRecommendationUpserter
	PrecomputedRecommendationDeleter
}

// PrecomputedRecommendationStore combines all precomputed recommendation operations.
type PrecomputedRecommendationStore interface {
	PrecomputedRecommendationReader
	PrecomputedRecommendationWriter
}

// UserRecommendationState represents the recommendation generation state for a user.
type UserRecommendationState struct {
	LastGeneratedAt   time.Time
	LastRatingAt      time.Time
	NeedsRegeneration bool
}

// UserRecommendationStateGetter retrieves the recommendation state for a user.
type UserRecommendationStateGetter interface {
	GetUserRecommendationState(ctx context.Context, userID string) (UserRecommendationState, error)
}

// UserRegenerationNeededMarker marks a user as needing recommendation regeneration.
type UserRegenerationNeededMarker interface {
	MarkUserNeedsRegeneration(ctx context.Context, userID string) error
}

// UserRegeneratedMarker marks a user's recommendations as regenerated.
type UserRegeneratedMarker interface {
	MarkUserRegenerated(ctx context.Context, userID string) error
}

// UsersNeedingRegenerationLister returns user IDs that need recommendation regeneration.
type UsersNeedingRegenerationLister interface {
	ListUsersNeedingRegeneration(ctx context.Context) ([]string, error)
}

// UserRecommendationRegenerationStatusRepository tracks regeneration status for users.
type UserRecommendationRegenerationStatusRepository interface {
	UserRegeneratedMarker
	UsersNeedingRegenerationLister
}

// UserRecommendationStateStore combines all recommendation state operations.
type UserRecommendationStateStore interface {
	UserRecommendationStateGetter
	UserRegenerationNeededMarker
	UserRecommendationRegenerationStatusRepository
}
