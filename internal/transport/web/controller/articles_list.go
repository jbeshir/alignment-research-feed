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
	Dataset     datasources.DatasetRepository
	CacheMaxAge time.Duration
}

func (c ArticlesList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters, err := articleFiltersFromQuery(r.URL.Query())
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse article filters in query string", "error", err)

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	articles, err := c.Dataset.ListLatestArticles(r.Context(), filters, 100)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch articles for feed", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(c.CacheMaxAge.Seconds())))

	if err := json.NewEncoder(w).Encode(articles); err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to write articles to response", "error", err)
	}
}
