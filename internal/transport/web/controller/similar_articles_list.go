package controller

import (
	"encoding/json"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
	"time"
)

type SimilarArticlesList struct {
	Fetcher     datasources.ArticleFetcher
	Similarity  datasources.SimilarArticleLister
	CacheMaxAge time.Duration
}

func (c SimilarArticlesList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("article_id")

	similarArticles, err := c.Similarity.ListSimilarArticles(r.Context(), articleID, 10)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch similar articles", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ids := make([]string, 0, len(similarArticles))
	for _, similar := range similarArticles {
		ids = append(ids, similar.HashID)
	}

	articles, err := c.Fetcher.FetchArticlesByID(r.Context(), ids)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch articles", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization")
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
