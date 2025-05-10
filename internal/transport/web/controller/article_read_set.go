package controller

import (
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
)

type ArticleReadSet struct {
	ReadSetter datasources.ArticleReadSetter
}

func (c ArticleReadSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]

	var read bool
	switch vars["read"] {
	case "true":
		read = true
	case "false":
		read = false
	default:
		logger := domain.LoggerFromContext(r.Context())
		logger.ErrorContext(r.Context(), "invalid read status", "read_status", vars["read"])
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := c.ReadSetter.SetArticleRead(r.Context(), id, domain.UserIDFromContext(r.Context()), read)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to mark article read", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
