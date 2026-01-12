package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type ArticleThumbsUpSet struct {
	Fetcher         datasources.ArticleFetcher
	ThumbsUpSetter  datasources.ArticleThumbsUpSetter
	AddVectorCmd    *command.AddArticleToUserVector
	RemoveVectorCmd *command.RemoveArticleFromUserVector
}

func (c ArticleThumbsUpSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", articleID))

	var thumbsUp bool
	switch vars["thumbs_up"] {
	case boolTrue:
		thumbsUp = true
	case boolFalse:
		thumbsUp = false
	default:
		logger.ErrorContext(ctx, "invalid thumbs_up value", "value", vars["thumbs_up"])
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
	err = c.ThumbsUpSetter.SetArticleThumbsUp(ctx, articleID, userID, thumbsUp)
	if err != nil {
		logger.ErrorContext(ctx, "unable to set thumbs up", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Sync vector after setting thumbs up
	if thumbsUp {
		if err := c.AddVectorCmd.Execute(ctx, userID, articleID); err != nil {
			logger.ErrorContext(ctx, "failed to add article vector to user", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		if err := c.RemoveVectorCmd.Execute(ctx, userID, articleID); err != nil {
			logger.ErrorContext(ctx, "failed to remove article vector from user", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
