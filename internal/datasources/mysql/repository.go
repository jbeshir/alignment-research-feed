package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql/queries"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"time"
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

func New(db *sql.DB) *Repository {
	return &Repository{db: db, queries: queries.New(db)}
}

func (r *Repository) ListLatestArticles(
	ctx context.Context,
	filters domain.ArticleFilters,
	options domain.ArticleListOptions,
) ([]domain.Article, error) {
	userID := domain.UserIDFromContext(ctx)

	sb := sqlbuilder.Select(
		"hash_id", "title", "url", "source",
		"LEFT(COALESCE(text, ''), 500) as text_start",
		"authors", "date_published", "have_read", "thumbs_up", "thumbs_down")
	sb.From("articles")
	sb.JoinWithOption(sqlbuilder.LeftJoin, "article_ratings",
		"articles.hash_id = article_ratings.article_hash_id",
		"article_ratings.user_id = "+sb.Args.Add(userID))

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
	defer rows.Close()

	articles := []domain.Article{}
	for rows.Next() {
		var i domain.Article
		var title sql.NullString
		var url sql.NullString
		var source sql.NullString
		var datePublished sql.NullTime
		var haveRead sql.NullBool
		var thumbsUp sql.NullBool
		var thumbsDown sql.NullBool

		if err := rows.Scan(
			&i.HashID,
			&title,
			&url,
			&source,
			&i.TextStart,
			&i.Authors,
			&datePublished,
			&haveRead,
			&thumbsUp,
			&thumbsDown,
		); err != nil {
			return nil, fmt.Errorf("scanning articles: %w", err)
		}

		// Just send nulls through as zero values
		i.Title = title.String
		i.Link = url.String
		i.Source = source.String
		i.PublishedAt = datePublished.Time
		if haveRead.Valid {
			i.HaveRead = &haveRead.Bool
		}
		if thumbsUp.Valid {
			i.ThumbsUp = &thumbsUp.Bool
		}
		if thumbsDown.Valid {
			i.ThumbsDown = &thumbsDown.Bool
		}

		articles = append(articles, i)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("closing rows iterator: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return articles, nil
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

	articles := make([]domain.Article, 0, len(dbArticles))
	for _, dbArticle := range dbArticles {
		articles = append(articles, domain.Article{
			HashID:      dbArticle.HashID,
			Title:       dbArticle.Title.String,
			Link:        dbArticle.Url.String,
			TextStart:   dbArticle.TextStart,
			Authors:     dbArticle.Authors,
			Source:      dbArticle.Source.String,
			PublishedAt: dbArticle.DatePublished.Time,
		})
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
