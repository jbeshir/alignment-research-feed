package controller

import (
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
)

type ArticleReadSet struct {
	Fetcher    datasources.ArticleFetcher
	ReadSetter datasources.ArticleReadSetter
}

func (c ArticleReadSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", id))

	var read bool
	switch vars["read"] {
	case "true":
		read = true
	case "false":
		read = false
	default:
		logger.ErrorContext(r.Context(), "invalid read status", "read_status", vars["read"])
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := c.Fetcher.FetchArticlesByID(ctx, []string{id})
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.ReadSetter.SetArticleRead(ctx, id, domain.UserIDFromContext(r.Context()), read)
	if err != nil {
		logger.ErrorContext(ctx, "unable to mark article read", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
