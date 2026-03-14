package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type ArticleGet struct {
	Fetcher     datasources.ArticleFetcher
	CacheMaxAge time.Duration
}

func (c ArticleGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]

	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	articles, err := c.Fetcher.FetchArticlesByID(ctx, []string{id})
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(articles) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if domain.UserIDFromContext(ctx) == "" {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(c.CacheMaxAge.Seconds())))
	}

	if err := json.NewEncoder(w).Encode(articles[0]); err != nil {
		logger.ErrorContext(ctx, "unable to write articles to response", "error", err)
	}
}
