package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type ArticleThumbsDownSet struct {
	Fetcher          datasources.ArticleFetcher
	ThumbsDownSetter datasources.ArticleThumbsDownSetter
	RemoveVectorCmd  *command.RemoveArticleFromUserVector
}

func (c ArticleThumbsDownSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", articleID))

	var thumbsDown bool
	switch vars["thumbs_down"] {
	case boolTrue:
		thumbsDown = true
	case boolFalse:
		thumbsDown = false
	default:
		logger.ErrorContext(ctx, "invalid thumbs_down value", "value", vars["thumbs_down"])
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := c.Fetcher.FetchArticlesByID(ctx, []string{articleID})
	if err != nil {
		logger.ErrorContext(ctx, "unable to fetch article", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userID := domain.UserIDFromContext(r.Context())
	err = c.ThumbsDownSetter.SetArticleThumbsDown(ctx, articleID, userID, thumbsDown)
	if err != nil {
		logger.ErrorContext(ctx, "unable to set thumbs down", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// When thumbs_down is set to true, it also clears thumbs_up (via SQL).
	// So we need to remove the vector if it was previously added.
	if thumbsDown {
		if err := c.RemoveVectorCmd.Execute(ctx, userID, articleID); err != nil {
			logger.ErrorContext(ctx, "failed to remove article vector from user", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
