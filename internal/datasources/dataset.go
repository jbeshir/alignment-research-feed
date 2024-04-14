package datasources

import (
	"context"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type DatasetRepository interface {
	ArticleLister
}

type ArticleLister interface {
	ListLatestArticles(ctx context.Context, limit int) ([]domain.Article, error)
}