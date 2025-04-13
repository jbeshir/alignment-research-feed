package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
	"time"
)

type ArticleGet struct {
	Fetcher     datasources.ArticleFetcher
	CacheMaxAge time.Duration
}

func (c ArticleGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]

	articles, err := c.Fetcher.FetchArticlesByID(r.Context(), []string{id})
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(c.CacheMaxAge.Seconds())))

	if err := json.NewEncoder(w).Encode(articles[0]); err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to write articles to response", "error", err)
	}
}
