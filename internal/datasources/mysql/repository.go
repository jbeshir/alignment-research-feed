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

func (r *Repository) ListLatestArticles(ctx context.Context, limit int) ([]domain.Article, error) {
	dbArticles, err := r.queries.ListLatestArticles(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("listing latest articles: %w", err)
	}

	articles := []domain.Article{}
	for _, article := range dbArticles {
		articles = append(articles, domain.Article{
			HashID:    article.HashID,
			Title:     article.Title.String,
			Link:      article.Url.String,
			TextStart: article.TextStart,
			Authors:   article.Authors,
			Published: article.DatePublished.Time,
		})
	}

	return articles, nil
}
