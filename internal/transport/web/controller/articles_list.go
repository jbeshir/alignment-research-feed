package controller

import (
	"encoding/json"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
	"time"
)

type ArticlesList struct {
	Lister      datasources.LatestArticleLister
	CacheMaxAge time.Duration
}

type ArticlesListResponse struct {
	Data     []domain.Article     `json:"data"`
	Metadata ArticlesListMetadata `json:"metadata"`
}

type ArticlesListMetadata struct{}

func (c ArticlesList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters, err := articleFiltersFromQuery(r.URL.Query())
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse article filters in query string", "error", err)

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	options, err := listOptionsFromQuery(r.URL.Query())
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse article list options in query string", "error", err)

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	articles, err := c.Lister.ListLatestArticles(r.Context(), filters, options)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch articles", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(c.CacheMaxAge.Seconds())))

	if err := json.NewEncoder(w).Encode(ArticlesListResponse{
		Data:     articles,
		Metadata: ArticlesListMetadata{},
	}); err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to write articles to response", "error", err)
	}
}
