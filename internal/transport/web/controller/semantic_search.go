package controller

import (
	"encoding/json"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

const maxTextBytes = 100 * 1024 // 100KB

type SemanticSearch struct {
	Embedder   datasources.Embedder
	Similarity datasources.SimilarArticlesByVectorLister
	Fetcher    datasources.ArticleFetcher
}

type semanticSearchRequest struct {
	Text  string `json:"text"`
	Limit int    `json:"limit"`
}

func (c SemanticSearch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	var req semanticSearchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxTextBytes+1024)).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(req.Text) > maxTextBytes {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	vector, err := c.Embedder.EmbedText(ctx, req.Text)
	if err != nil {
		logger.ErrorContext(ctx, "unable to embed text", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if vector == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	similarArticles, err := c.Similarity.ListSimilarArticlesByVector(ctx, nil, vector, limit)
	if err != nil {
		logger.ErrorContext(ctx, "unable to find similar articles", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ids := make([]string, 0, len(similarArticles))
	for _, similar := range similarArticles {
		ids = append(ids, similar.HashID)
	}

	articles, err := c.Fetcher.FetchArticlesByID(ctx, ids)
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch articles", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ArticlesListResponse{
		Data:     articles,
		Metadata: ArticlesListMetadata{},
	}); err != nil {
		logger.ErrorContext(ctx, "unable to write articles to response", "error", err)
	}
}
