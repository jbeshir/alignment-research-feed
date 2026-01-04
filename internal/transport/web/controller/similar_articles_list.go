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

type SimilarArticlesList struct {
	Fetcher     datasources.ArticleFetcher
	Similarity  datasources.SimilarArticleLister
	CacheMaxAge time.Duration
}

func (c SimilarArticlesList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID := vars["article_id"]

	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	if articleID == "" {
		logger.ErrorContext(ctx, "article_id not set in request")

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	similarArticles, err := c.Similarity.ListSimilarArticles(ctx, []string{articleID}, 10)
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch similar articles", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ids := make([]string, 0, len(similarArticles))
	for _, similar := range similarArticles {
		ids = append(ids, similar.HashID)
	}

	articles, err := c.Fetcher.FetchArticlesByID(ctx, ids)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch articles", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if domain.UserIDFromContext(r.Context()) == "" {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(c.CacheMaxAge.Seconds())))
	}

	if err := json.NewEncoder(w).Encode(ArticlesListResponse{
		Data:     articles,
		Metadata: ArticlesListMetadata{},
	}); err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to write articles to response", "error", err)
	}
}
