package controller

import (
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
)

type ArticleMarkRead struct {
	ReadMarker datasources.ArticleReadMarker
}

func (c ArticleMarkRead) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["article_id"]

	err := c.ReadMarker.MarkArticleRead(r.Context(), id, domain.UserIDFromContext(r.Context()))
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to mark article read", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
