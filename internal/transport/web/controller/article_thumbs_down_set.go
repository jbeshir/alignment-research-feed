package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type ArticleThumbsDownSet struct {
	Fetcher          datasources.ArticleFetcher
	ThumbsDownSetter datasources.ArticleThumbsDownSetter
}

func (c ArticleThumbsDownSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", id))

	var thumbsDown bool
	switch vars["thumbs_down"] {
	case "true":
		thumbsDown = true
	case "false":
		thumbsDown = false
	default:
		logger.ErrorContext(r.Context(), "invalid thumbs_down status", "thumbs_down_status", vars["thumbs_down"])
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := c.Fetcher.FetchArticlesByID(ctx, []string{id})
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.ThumbsDownSetter.SetArticleThumbsDown(ctx, id, domain.UserIDFromContext(r.Context()), thumbsDown)
	if err != nil {
		logger.ErrorContext(ctx, "unable to mark article thumbs_down", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
