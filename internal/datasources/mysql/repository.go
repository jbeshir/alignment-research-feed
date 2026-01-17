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
		ArticleHashID: hashID,
		UserID:        userID,
		HaveRead:      sql.NullBool{Bool: read, Valid: true},
		DateRead:      dateRead,
	})
}

func (r *Repository) SetArticleThumbsUp(ctx context.Context, hashID, userID string, thumbsUp bool) error {
	var dateReviewed sql.NullTime
	if thumbsUp {
		dateReviewed = sql.NullTime{Time: time.Now(), Valid: true}
	}
	return r.queries.SetArticleThumbsUp(ctx, queries.SetArticleThumbsUpParams{
		ArticleHashID: hashID,
		UserID:        userID,
		ThumbsUp:      sql.NullBool{Bool: thumbsUp, Valid: true},
		DateReviewed:  dateReviewed,
	})
}

func (r *Repository) SetArticleThumbsDown(ctx context.Context, hashID, userID string, thumbsDown bool) error {
	var dateReviewed sql.NullTime
	if thumbsDown {
		dateReviewed = sql.NullTime{Time: time.Now(), Valid: true}
	}
	return r.queries.SetArticleThumbsDown(ctx, queries.SetArticleThumbsDownParams{
		ArticleHashID: hashID,
		UserID:        userID,
		ThumbsDown:    sql.NullBool{Bool: thumbsDown, Valid: true},
		DateReviewed:  dateReviewed,
	})
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

// GetUserVector retrieves the stored vector sum and count for a user.
func (r *Repository) GetUserVector(ctx context.Context, userID string) ([]float32, int, error) {
	return getUserVectorWithQueries(ctx, r.queries, userID)
}

// AddArticleVectorToUser atomically checks if the article's vector has already been added,
// and if not, adds it to the user's vector sum and marks it as added.
// Returns true if the vector was added, false if it was already added.
func (r *Repository) AddArticleVectorToUser(
	ctx context.Context, userID, articleHashID string, vector []float32,
) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := r.queries.WithTx(tx)

	// Check if vector was already added
	added, err := qtx.GetRatingVectorAdded(ctx, queries.GetRatingVectorAddedParams{
		UserID:        userID,
		ArticleHashID: articleHashID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("checking vector added status: %w", err)
	}
	if added {
		return false, nil // Already added
	}

	// Get current vector sum
	currentSum, _, err := getUserVectorWithQueries(ctx, qtx, userID)
	if err != nil {
		return false, fmt.Errorf("getting current user vector: %w", err)
	}

	var newSum []float32
	if currentSum == nil {
		newSum = vector
	} else {
		newSum, err = addVectors(currentSum, vector)
		if err != nil {
			return false, fmt.Errorf("adding vectors: %w", err)
		}
	}

	newSumBytes := float32SliceToBytes(newSum)
	if err := qtx.UpsertUserVectorAdd(ctx, queries.UpsertUserVectorAddParams{
		UserID:      userID,
		VectorSum:   newSumBytes,
		VectorSum_2: newSumBytes,
	}); err != nil {
		return false, fmt.Errorf("upserting user vector: %w", err)
	}

	// Mark vector as added
	if err := qtx.SetRatingVectorAdded(ctx, queries.SetRatingVectorAddedParams{
		VectorAdded:   true,
		UserID:        userID,
		ArticleHashID: articleHashID,
	}); err != nil {
		return false, fmt.Errorf("marking vector as added: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing transaction: %w", err)
	}

	return true, nil
}

// SubtractArticleVectorFromUser atomically checks if the article's vector was previously added,
// and if so, subtracts it from the user's vector sum and marks it as removed.
// If vector is nil, only clears the added flag without modifying the sum.
// Returns true if the flag was cleared, false if it wasn't set.
func (r *Repository) SubtractArticleVectorFromUser(
	ctx context.Context, userID, articleHashID string, vector []float32,
) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := r.queries.WithTx(tx)

	// Check if vector was added
	added, err := qtx.GetRatingVectorAdded(ctx, queries.GetRatingVectorAddedParams{
		UserID:        userID,
		ArticleHashID: articleHashID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("checking vector added status: %w", err)
	}
	if !added {
		return false, nil // Wasn't added
	}

	// Subtract vector from sum if provided
	if vector != nil {
		currentSum, _, err := getUserVectorWithQueries(ctx, qtx, userID)
		if err != nil {
			return false, fmt.Errorf("getting current user vector: %w", err)
		}

		if currentSum != nil {
			newSum, err := subtractVectors(currentSum, vector)
			if err != nil {
				return false, fmt.Errorf("subtracting vectors: %w", err)
			}
			newSumBytes := float32SliceToBytes(newSum)

			if err := qtx.UpdateUserVectorSubtract(ctx, queries.UpdateUserVectorSubtractParams{
				VectorSum: newSumBytes,
				UserID:    userID,
			}); err != nil {
				return false, fmt.Errorf("updating user vector: %w", err)
			}
		}
	}

	// Mark vector as removed
	if err := qtx.SetRatingVectorAdded(ctx, queries.SetRatingVectorAddedParams{
		VectorAdded:   false,
		UserID:        userID,
		ArticleHashID: articleHashID,
	}); err != nil {
		return false, fmt.Errorf("marking vector as removed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing transaction: %w", err)
	}

	return true, nil
}

// getUserVectorWithQueries retrieves the user vector sum and count using the provided queries object.
func getUserVectorWithQueries(
	ctx context.Context, q *queries.Queries, userID string,
) ([]float32, int, error) {
	row, err := q.GetUserVector(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("fetching user vector: %w", err)
	}

	vector, err := bytesToFloat32Slice(row.VectorSum)
	if err != nil {
		return nil, 0, fmt.Errorf("decoding user vector: %w", err)
	}

	return vector, int(row.VectorCount), nil
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

func addVectors(a, b []float32) ([]float32, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("vector length mismatch: %d vs %d", len(a), len(b))
	}
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result, nil
}

func subtractVectors(a, b []float32) ([]float32, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("vector length mismatch: %d vs %d", len(a), len(b))
	}
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result, nil
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
