package mysql

import (
	"context"
	"database/sql"
	"fmt"
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
	return r.queries.SetArticleRead(ctx, queries.SetArticleReadParams{
		ArticleHashID: hashID,
		UserID:        userID,
		HaveRead:      sql.NullBool{Bool: read, Valid: true},
	})
}

func (r *Repository) SetArticleThumbsUp(ctx context.Context, hashID, userID string, thumbsUp bool) error {
	return r.queries.SetArticleThumbsUp(ctx, queries.SetArticleThumbsUpParams{
		ArticleHashID: hashID,
		UserID:        userID,
		ThumbsUp:      sql.NullBool{Bool: thumbsUp, Valid: true},
	})
}

func (r *Repository) SetArticleThumbsDown(ctx context.Context, hashID, userID string, thumbsDown bool) error {
	return r.queries.SetArticleThumbsDown(ctx, queries.SetArticleThumbsDownParams{
		ArticleHashID: hashID,
		UserID:        userID,
		ThumbsDown:    sql.NullBool{Bool: thumbsDown, Valid: true},
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
