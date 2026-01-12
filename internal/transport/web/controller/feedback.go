package controller

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// Bool string constants for route parameters.
const (
	boolTrue  = "true"
	boolFalse = "false"
)

type feedbackSetter func(ctx context.Context, hashID string, userID string, value bool) error

func handleFeedback(
	w http.ResponseWriter,
	r *http.Request,
	fetcher datasources.ArticleFetcher,
	setter feedbackSetter,
	paramName string,
) {
	vars := mux.Vars(r)
	id := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", id))

	var value bool
	switch vars[paramName] {
	case boolTrue:
		value = true
	case boolFalse:
		value = false
	default:
		logger.ErrorContext(r.Context(), "invalid status", "status", vars[paramName])
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := fetcher.FetchArticlesByID(ctx, []string{id})
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = setter(ctx, id, domain.UserIDFromContext(r.Context()), value)
	if err != nil {
		logger.ErrorContext(ctx, "unable to set feedback", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
