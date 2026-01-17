package controller

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// UserArticlesLister is a function type that lists article IDs for a user with pagination.
type UserArticlesLister func(ctx context.Context, userID string, page, pageSize int) ([]string, error)

// UserArticlesList is a generic controller for user-specific article lists.
type UserArticlesList struct {
	Fetcher    datasources.ArticleFetcher
	ListFunc   UserArticlesLister
	ListEntity string // For error messages, e.g., "unreviewed", "liked", "disliked"
}

func (c UserArticlesList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	userID := domain.UserIDFromContext(ctx)
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	page, pageSize, err := parsePagination(r.URL.Query())
	if err != nil {
		logger.ErrorContext(ctx, "unable to parse pagination", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	articleIDs, err := c.ListFunc(ctx, userID, page, pageSize)
	if err != nil {
		logger.ErrorContext(ctx, "unable to list "+c.ListEntity+" article IDs", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	articles, err := c.Fetcher.FetchArticlesByID(ctx, articleIDs)
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article metadata", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(ArticlesListResponse{
		Data:     articles,
		Metadata: ArticlesListMetadata{},
	}); err != nil {
		logger.ErrorContext(ctx, "unable to write "+c.ListEntity+" articles to response", "error", err)
	}
}
