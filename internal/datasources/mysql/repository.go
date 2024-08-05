package mysql

import (
	"context"
	"database/sql"
	"fmt"
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

func New(db *sql.DB) *Repository {
	return &Repository{db: db, queries: queries.New(db)}
}

func (r *Repository) ListLatestArticles(
	ctx context.Context,
	filters domain.ArticleFilters,
	options domain.ArticleListOptions,
) ([]domain.Article, error) {

	sb := sqlbuilder.Select(
		"hash_id", "title", "url", "source",
		"LEFT(COALESCE(text, ''), 500) as text_start",
		"authors", "date_published")
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
	sb.Offset(options.Page - 1*options.PageSize)
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

		if err := rows.Scan(
			&i.HashID,
			&title,
			&url,
			&source,
			&i.TextStart,
			&i.Authors,
			&datePublished,
		); err != nil {
			return nil, fmt.Errorf("scanning articleL %w", err)
		}

		// Just send nulls through as zero values
		i.Title = title.String
		i.Link = url.String
		i.Source = source.String
		i.PublishedAt = datePublished.Time

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
	if len(filters.OnlySources) > 0 {
		onlySources := make([]interface{}, 0, len(filters.OnlySources))
		for _, source := range filters.OnlySources {
			onlySources = append(onlySources, source)
		}

		cond := sb.In("source", onlySources...)
		conds = append(conds, cond)
	}

	if len(filters.ExceptSources) > 0 {
		exceptSources := make([]interface{}, 0, len(filters.ExceptSources))
		for _, source := range filters.ExceptSources {
			exceptSources = append(exceptSources, source)
		}

		cond := sb.NotIn("source", exceptSources...)
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
