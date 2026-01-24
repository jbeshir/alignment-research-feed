package mysql

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql/queries"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

//go:generate sqlc generate

var _ datasources.DatasetRepository = (*Repository)(nil)

type Repository struct {
	db      *sql.DB
	queries *queries.Queries
}

func (r *Repository) SetArticleRead(ctx context.Context, hashID, userID string, read bool) error {
	var dateRead sql.NullTime
	if read {
		dateRead = sql.NullTime{Time: time.Now(), Valid: true}
	}
	return r.queries.SetArticleRead(ctx, queries.SetArticleReadParams{
		UserID:        userID,
		ArticleHashID: hashID,
		HaveRead:      read,
		DateRead:      dateRead,
	})
}

// interactionState captures the current state of a user-article interaction.
type interactionState struct {
	currentHaveRead bool
	currentDateRead sql.NullTime
}

// SetArticleRating atomically sets thumbs up/down.
func (r *Repository) SetArticleRating(
	ctx context.Context, userID, articleHashID string,
	thumbsUp, thumbsDown bool, vector []float32,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := r.queries.WithTx(tx)

	state, err := r.getCurrentInteractionState(ctx, qtx, userID, articleHashID)
	if err != nil {
		return err
	}

	if err := r.upsertInteraction(ctx, qtx, userID, articleHashID, thumbsUp, thumbsDown, vector, state); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// getCurrentInteractionState fetches the current state of a user-article interaction.
func (r *Repository) getCurrentInteractionState(
	ctx context.Context, qtx *queries.Queries, userID, articleHashID string,
) (interactionState, error) {
	current, err := qtx.GetUserArticleInteraction(ctx, queries.GetUserArticleInteractionParams{
		UserID:        userID,
		ArticleHashID: articleHashID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return interactionState{}, nil
	}
	if err != nil {
		return interactionState{}, fmt.Errorf("getting current interaction: %w", err)
	}
	return interactionState{
		currentHaveRead: current.HaveRead,
		currentDateRead: current.DateRead,
	}, nil
}

// upsertInteraction stores the interaction record.
func (r *Repository) upsertInteraction(
	ctx context.Context, qtx *queries.Queries, userID, articleHashID string,
	thumbsUp, thumbsDown bool, vector []float32, state interactionState,
) error {
	var vectorStr sql.NullString
	if vector != nil {
		vectorStr = sql.NullString{String: string(float32SliceToBytes(vector)), Valid: true}
	}

	var dateRated sql.NullTime
	if thumbsUp || thumbsDown {
		dateRated = sql.NullTime{Time: time.Now(), Valid: true}
	}

	if err := qtx.UpsertUserArticleInteraction(ctx, queries.UpsertUserArticleInteractionParams{
		UserID:        userID,
		ArticleHashID: articleHashID,
		HaveRead:      state.currentHaveRead,
		ThumbsUp:      thumbsUp,
		ThumbsDown:    thumbsDown,
		DateRead:      state.currentDateRead,
		DateRated:     dateRated,
		Vector:        vectorStr,
	}); err != nil {
		return fmt.Errorf("upserting interaction: %w", err)
	}
	return nil
}

func (r *Repository) ListThumbsUpArticleIDs(ctx context.Context, userID string) ([]string, error) {
	return r.queries.ListThumbsUpArticleIDs(ctx, userID)
}

func (r *Repository) ListUnreviewedArticleIDs(
	ctx context.Context, userID string, page, pageSize int,
) ([]string, error) {
	limit, offset := paginationToLimitOffset(page, pageSize)
	return r.queries.ListUnreviewedArticleIDs(ctx, queries.ListUnreviewedArticleIDsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (r *Repository) ListLikedArticleIDs(
	ctx context.Context, userID string, page, pageSize int,
) ([]string, error) {
	limit, offset := paginationToLimitOffset(page, pageSize)
	return r.queries.ListLikedArticleIDs(ctx, queries.ListLikedArticleIDsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (r *Repository) ListDislikedArticleIDs(
	ctx context.Context, userID string, page, pageSize int,
) ([]string, error) {
	limit, offset := paginationToLimitOffset(page, pageSize)
	return r.queries.ListDislikedArticleIDs(ctx, queries.ListDislikedArticleIDsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (r *Repository) ListReadArticleIDs(ctx context.Context, userID string) ([]string, error) {
	return r.queries.ListReadArticleIDs(ctx, userID)
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db, queries: queries.New(db)}
}

func (r *Repository) ListLatestArticleIDs(
	ctx context.Context,
	filters domain.ArticleFilters,
	options domain.ArticleListOptions,
) ([]string, error) {
	sb := sqlbuilder.Select("hash_id")
	sb.From("articles")

	conds := buildArticlesConditions(sb, filters)
	if len(conds) > 0 {
		sb.Where(conds...)
	}

	orderings, err := buildArticlesOrder(options)
	if err != nil {
		return nil, fmt.Errorf("building articles order by clause: %w", err)
	}

	sb.OrderBy(orderings...)
	sb.Offset((options.Page - 1) * options.PageSize)
	sb.Limit(options.PageSize)

	query, args := sb.Build()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("running articles query: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Log the error but don't override the main error
			_ = closeErr // Explicitly ignore the error
		}
	}()

	articleIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(
			&id,
		); err != nil {
			return nil, fmt.Errorf("scanning articles: %w", err)
		}
		articleIDs = append(articleIDs, id)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("closing rows iterator: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return articleIDs, nil
}

func (r *Repository) FetchArticlesByID(
	ctx context.Context,
	hashIDs []string,
) ([]domain.Article, error) {

	dbArticles, err := r.queries.FetchArticlesByID(ctx, queries.FetchArticlesByIDParams{
		HashIds: hashIDs,
		UserID:  domain.UserIDFromContext(ctx),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching articles by ID: %w", err)
	}

	// Create a map for quick lookup of articles by hash_id
	articleMap := make(map[string]domain.Article, len(dbArticles))
	for _, dbArticle := range dbArticles {
		var haveRead, thumbsUp, thumbsDown *bool

		if dbArticle.HaveRead.Valid {
			haveRead = &dbArticle.HaveRead.Bool
		}
		if dbArticle.ThumbsUp.Valid {
			thumbsUp = &dbArticle.ThumbsUp.Bool
		}
		if dbArticle.ThumbsDown.Valid {
			thumbsDown = &dbArticle.ThumbsDown.Bool
		}

		articleMap[dbArticle.HashID] = domain.Article{
			HashID:      dbArticle.HashID,
			Title:       dbArticle.Title.String,
			Link:        dbArticle.Url.String,
			TextStart:   dbArticle.TextStart,
			Authors:     dbArticle.Authors,
			Source:      dbArticle.Source.String,
			PublishedAt: dbArticle.DatePublished.Time,
			HaveRead:    haveRead,
			ThumbsUp:    thumbsUp,
			ThumbsDown:  thumbsDown,
		}
	}

	// Build results in the same order as the input hashIDs
	articles := make([]domain.Article, 0, len(hashIDs))
	for _, hashID := range hashIDs {
		if article, exists := articleMap[hashID]; exists {
			articles = append(articles, article)
		}
	}

	return articles, nil
}

func (r *Repository) TotalMatchingArticles(
	ctx context.Context,
	filters domain.ArticleFilters,
) (int64, error) {

	sb := sqlbuilder.Select("COUNT(*)")
	sb.From("articles")

	conds := buildArticlesConditions(sb, filters)
	if len(conds) > 0 {
		sb.Where(conds...)
	}

	query, queryParams := sb.Build()

	row := r.db.QueryRowContext(ctx, query, queryParams...)
	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting matching articles: %w", err)
	}
	return count, err
}

func buildArticlesConditions(sb *sqlbuilder.SelectBuilder, filters domain.ArticleFilters) []string {
	var conds []string

	if filters.TitleFulltext != "" {
		conds = append(conds, "MATCH (title) AGAINST ("+sb.Args.Add(filters.TitleFulltext)+")")
	}

	if filters.AuthorsFulltext != "" {
		conds = append(conds, "MATCH (authors) AGAINST ("+sb.Args.Add(filters.AuthorsFulltext)+")")
	}

	if filters.PublishedAfter != (time.Time{}) {
		conds = append(conds, sb.GreaterEqualThan("date_published", filters.PublishedAfter))
	}

	if filters.PublishedBefore != (time.Time{}) {
		conds = append(conds, sb.LessEqualThan("date_published", filters.PublishedBefore))
	}

	if len(filters.SourcesAllowlist) > 0 {
		allowed := make([]interface{}, 0, len(filters.SourcesAllowlist))
		for _, source := range filters.SourcesAllowlist {
			allowed = append(allowed, source)
		}

		cond := sb.In("source", allowed...)
		conds = append(conds, cond)
	}

	if len(filters.SourcesBlocklist) > 0 {
		blocked := make([]interface{}, 0, len(filters.SourcesBlocklist))
		for _, source := range filters.SourcesBlocklist {
			blocked = append(blocked, source)
		}

		cond := sb.NotIn("source", blocked...)
		conds = append(conds, cond)
	}

	return conds
}

func buildArticlesOrder(options domain.ArticleListOptions) ([]string, error) {
	if len(options.Ordering) == 0 {
		return []string{"date_published DESC"}, nil
	}

	var orderings []string
	for _, ordering := range options.Ordering {
		var col string
		switch ordering.Field {
		case domain.ArticleOrderingFieldAuthors:
			col = "authors"
		case domain.ArticleOrderingFieldPublishedAt:
			col = "date_published"
		case domain.ArticleOrderingFieldSource:
			col = "source"
		case domain.ArticleOrderingFieldTitle:
			col = "title"
		default:
			return nil, fmt.Errorf("unknown ordering field: %s", ordering.Field)
		}

		if ordering.Desc {
			col += " DESC"
		}
		orderings = append(orderings, col)
	}

	return orderings, nil
}

// Helper functions for binary vector serialization

func float32SliceToBytes(floats []float32) []byte {
	bytes := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(f))
	}
	return bytes
}

func bytesToFloat32Slice(bytes []byte) ([]float32, error) {
	if len(bytes)%4 != 0 {
		return nil, fmt.Errorf("invalid byte length for float32 slice: %d", len(bytes))
	}
	floats := make([]float32, len(bytes)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(bytes[i*4:]))
	}
	return floats, nil
}

// paginationToLimitOffset converts page/pageSize to limit/offset with bounds checking.
// Clamps values to int32 range to prevent overflow.
func paginationToLimitOffset(page, pageSize int) (limit, offset int32) {
	if pageSize > math.MaxInt32 {
		pageSize = math.MaxInt32
	}
	limit = int32(pageSize) //nolint:gosec // bounds checked above

	off := (page - 1) * pageSize
	if off > math.MaxInt32 {
		off = math.MaxInt32
	}
	offset = int32(off) //nolint:gosec // bounds checked above

	return limit, offset
}

// ============================================
// User Article Interaction Store Implementation
// ============================================

// GetUserArticleVectorsByType retrieves article vectors for a user filtered by rating type.
func (r *Repository) GetUserArticleVectorsByType(
	ctx context.Context, userID string, ratingType domain.UserRatingType,
) ([]domain.UserArticleRating, error) {
	switch ratingType {
	case domain.RatingTypeThumbsUp:
		rows, err := r.queries.GetUserArticleVectorsByThumbsUp(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("fetching thumbs up vectors: %w", err)
		}
		return convertVectorRows(rows, domain.RatingTypeThumbsUp)

	case domain.RatingTypeThumbsDown:
		rows, err := r.queries.GetUserArticleVectorsByThumbsDown(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("fetching thumbs down vectors: %w", err)
		}
		return convertVectorRowsDown(rows, domain.RatingTypeThumbsDown)

	default:
		return nil, fmt.Errorf("unknown rating type: %s", ratingType)
	}
}

func convertVectorRows(
	rows []queries.GetUserArticleVectorsByThumbsUpRow,
	ratingType domain.UserRatingType,
) ([]domain.UserArticleRating, error) {
	result := make([]domain.UserArticleRating, 0, len(rows))
	for _, row := range rows {
		if !row.Vector.Valid {
			continue // Skip rows without vectors
		}
		vector, err := bytesToFloat32Slice([]byte(row.Vector.String))
		if err != nil {
			return nil, fmt.Errorf("decoding vector for article %s: %w", row.ArticleHashID, err)
		}
		result = append(result, domain.UserArticleRating{
			ArticleHashID: row.ArticleHashID,
			Vector:        vector,
			RatingType:    ratingType,
			RatedAt:       row.DateRated.Time,
		})
	}
	return result, nil
}

func convertVectorRowsDown(
	rows []queries.GetUserArticleVectorsByThumbsDownRow,
	ratingType domain.UserRatingType,
) ([]domain.UserArticleRating, error) {
	result := make([]domain.UserArticleRating, 0, len(rows))
	for _, row := range rows {
		if !row.Vector.Valid {
			continue // Skip rows without vectors
		}
		vector, err := bytesToFloat32Slice([]byte(row.Vector.String))
		if err != nil {
			return nil, fmt.Errorf("decoding vector for article %s: %w", row.ArticleHashID, err)
		}
		result = append(result, domain.UserArticleRating{
			ArticleHashID: row.ArticleHashID,
			Vector:        vector,
			RatingType:    ratingType,
			RatedAt:       row.DateRated.Time,
		})
	}
	return result, nil
}

// CountUserArticleVectorsByType returns the count of vectors for a user by rating type.
func (r *Repository) CountUserArticleVectorsByType(
	ctx context.Context, userID string, ratingType domain.UserRatingType,
) (int64, error) {
	switch ratingType {
	case domain.RatingTypeThumbsUp:
		return r.queries.CountUserArticleVectorsByThumbsUp(ctx, userID)
	case domain.RatingTypeThumbsDown:
		return r.queries.CountUserArticleVectorsByThumbsDown(ctx, userID)
	default:
		return 0, fmt.Errorf("unknown rating type: %s", ratingType)
	}
}

// ============================================
// User Interest Cluster Store Implementation
// ============================================

// UpsertUserInterestCluster stores or updates a user's interest cluster.
func (r *Repository) UpsertUserInterestCluster(
	ctx context.Context, userID string, clusterID int, centroidVector []float32, articleCount int,
) error {
	vectorBytes := float32SliceToBytes(centroidVector)
	return r.queries.UpsertUserInterestCluster(ctx, queries.UpsertUserInterestClusterParams{
		UserID:         userID,
		ClusterID:      int32(clusterID), //nolint:gosec // cluster IDs are small
		CentroidVector: vectorBytes,
		ArticleCount:   int32(articleCount), //nolint:gosec // article counts are bounded
	})
}

// GetUserInterestClusters retrieves all interest clusters for a user.
func (r *Repository) GetUserInterestClusters(
	ctx context.Context, userID string,
) ([]datasources.UserInterestCluster, error) {
	rows, err := r.queries.GetUserInterestClusters(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching user interest clusters: %w", err)
	}

	result := make([]datasources.UserInterestCluster, 0, len(rows))
	for _, row := range rows {
		vector, err := bytesToFloat32Slice(row.CentroidVector)
		if err != nil {
			return nil, fmt.Errorf("decoding centroid vector for cluster %d: %w", row.ClusterID, err)
		}
		result = append(result, datasources.UserInterestCluster{
			ClusterID:      int(row.ClusterID),
			CentroidVector: vector,
			ArticleCount:   int(row.ArticleCount),
			UpdatedAt:      row.UpdatedAt,
		})
	}

	return result, nil
}

// DeleteUserInterestClusters removes all interest clusters for a user.
func (r *Repository) DeleteUserInterestClusters(ctx context.Context, userID string) error {
	return r.queries.DeleteUserInterestClusters(ctx, userID)
}

// ============================================
// Precomputed Recommendation Store Implementation
// ============================================

// UpsertPrecomputedRecommendation stores or updates a precomputed recommendation.
func (r *Repository) UpsertPrecomputedRecommendation(
	ctx context.Context,
	userID, articleHashID string,
	score float64,
	source string,
	position int,
	generatedAt time.Time,
) error {
	return r.queries.UpsertPrecomputedRecommendation(ctx, queries.UpsertPrecomputedRecommendationParams{
		UserID:        userID,
		ArticleHashID: articleHashID,
		Score:         score,
		Source:        source,
		Position:      int32(position), //nolint:gosec // positions are small
		GeneratedAt:   generatedAt,
	})
}

// DeleteUserPrecomputedRecommendations removes all precomputed recommendations for a user.
func (r *Repository) DeleteUserPrecomputedRecommendations(ctx context.Context, userID string) error {
	return r.queries.DeleteUserPrecomputedRecommendations(ctx, userID)
}

// GetPrecomputedRecommendations retrieves precomputed recommendations for a user, ordered by position.
func (r *Repository) GetPrecomputedRecommendations(
	ctx context.Context, userID string, limit int,
) ([]datasources.PrecomputedRecommendation, error) {
	rows, err := r.queries.GetPrecomputedRecommendations(ctx, queries.GetPrecomputedRecommendationsParams{
		UserID: userID,
		Limit:  int32(limit), //nolint:gosec // limits are small
	})
	if err != nil {
		return nil, fmt.Errorf("fetching precomputed recommendations: %w", err)
	}

	result := make([]datasources.PrecomputedRecommendation, 0, len(rows))
	for _, row := range rows {
		result = append(result, datasources.PrecomputedRecommendation{
			ArticleHashID: row.ArticleHashID,
			Score:         row.Score,
			Source:        row.Source,
			Position:      int(row.Position),
			GeneratedAt:   row.GeneratedAt,
		})
	}

	return result, nil
}

// GetPrecomputedRecommendationAge returns when recommendations were last generated for a user.
// Returns zero time if no recommendations exist.
func (r *Repository) GetPrecomputedRecommendationAge(ctx context.Context, userID string) (time.Time, error) {
	generatedAt, err := r.queries.GetPrecomputedRecommendationAge(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("fetching recommendation age: %w", err)
	}
	return generatedAt, nil
}

// ============================================
// User Recommendation State Store Implementation
// ============================================

// GetUserRecommendationState retrieves the recommendation state for a user.
func (r *Repository) GetUserRecommendationState(
	ctx context.Context, userID string,
) (datasources.UserRecommendationState, error) {
	row, err := r.queries.GetUserRecommendationState(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return datasources.UserRecommendationState{}, nil
		}
		return datasources.UserRecommendationState{}, fmt.Errorf("fetching recommendation state: %w", err)
	}

	return datasources.UserRecommendationState{
		LastGeneratedAt:   row.LastGeneratedAt.Time,
		LastRatingAt:      row.LastRatingAt.Time,
		NeedsRegeneration: row.NeedsRegeneration,
	}, nil
}

// MarkUserNeedsRegeneration marks a user as needing recommendation regeneration.
func (r *Repository) MarkUserNeedsRegeneration(ctx context.Context, userID string) error {
	return r.queries.MarkUserNeedsRegeneration(ctx, userID)
}

// MarkUserRegenerated marks a user's recommendations as regenerated.
func (r *Repository) MarkUserRegenerated(ctx context.Context, userID string) error {
	return r.queries.MarkUserRegenerated(ctx, userID)
}

// ListUsersNeedingRegeneration returns user IDs that need recommendation regeneration.
func (r *Repository) ListUsersNeedingRegeneration(ctx context.Context) ([]string, error) {
	return r.queries.ListUsersNeedingRegeneration(ctx)
}
