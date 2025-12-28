package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type ArticleThumbsUpSet struct {
	Fetcher        datasources.ArticleFetcher
	ThumbsUpSetter datasources.ArticleThumbsUpSetter
}

func (c ArticleThumbsUpSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", id))

	var thumbsUp bool
	switch vars["thumbs_up"] {
	case "true":
		thumbsUp = true
	case "false":
		thumbsUp = false
	default:
		logger.ErrorContext(r.Context(), "invalid thumbs_up status", "thumbs_up_status", vars["thumbs_up"])
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := c.Fetcher.FetchArticlesByID(ctx, []string{id})
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.ThumbsUpSetter.SetArticleThumbsUp(ctx, id, domain.UserIDFromContext(r.Context()), thumbsUp)
	if err != nil {
		logger.ErrorContext(ctx, "unable to mark article thumbs_up", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
