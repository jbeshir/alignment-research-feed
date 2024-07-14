package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql/queries"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

//go:generate sqlc generate

var _ datasources.DatasetRepository = (*Repository)(nil)

type Repository struct {
	queries *queries.Queries
}

func New(db *sql.DB) *Repository {
	return &Repository{queries: queries.New(db)}
}

func (r *Repository) ListLatestArticles(
	ctx context.Context,
	filters domain.ArticleFilters,
	limit int,
) ([]domain.Article, error) {

	onlySources := make([]sql.NullString, 0, len(filters.OnlySources))
	for _, source := range filters.OnlySources {
		onlySources = append(onlySources, sql.NullString{String: source, Valid: true})
	}

	exceptSources := make([]sql.NullString, 0, len(filters.ExceptSources))
	for _, source := range filters.ExceptSources {
		exceptSources = append(exceptSources, sql.NullString{String: source, Valid: true})
	}

	dbArticles, err := r.queries.ListLatestArticles(ctx, queries.ListLatestArticlesParams{
		OnlySourcesFilter:   len(filters.OnlySources) > 0,
		OnlySources:         onlySources,
		ExceptSourcesFilter: len(filters.ExceptSources) > 0,
		ExceptSources:       exceptSources,
		Limit:               int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("listing latest articles: %w", err)
	}

	articles := []domain.Article{}
	for _, article := range dbArticles {
		articles = append(articles, domain.Article{
			HashID:      article.HashID,
			Title:       article.Title.String,
			Link:        article.Url.String,
			TextStart:   article.TextStart,
			Authors:     article.Authors,
			Source:      article.Source.String,
			PublishedAt: article.DatePublished.Time,
		})
	}

	return articles, nil
}

func (r *Repository) TotalMatchingArticles(
	ctx context.Context,
	filters domain.ArticleFilters,
) (int64, error) {

	onlySources := make([]sql.NullString, 0, len(filters.OnlySources))
	for _, source := range filters.OnlySources {
		onlySources = append(onlySources, sql.NullString{String: source, Valid: true})
	}

	exceptSources := make([]sql.NullString, 0, len(filters.ExceptSources))
	for _, source := range filters.ExceptSources {
		exceptSources = append(exceptSources, sql.NullString{String: source, Valid: true})
	}

	count, err := r.queries.TotalMatchingArticles(ctx, queries.TotalMatchingArticlesParams{
		OnlySourcesFilter:   len(filters.OnlySources) > 0,
		OnlySources:         onlySources,
		ExceptSourcesFilter: len(filters.ExceptSources) > 0,
		ExceptSources:       exceptSources,
	})
	if err != nil {
		return 0, fmt.Errorf("counting matching articles: %w", err)
	}

	return count, nil
}
