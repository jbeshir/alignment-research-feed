package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type ArticleRatingSet struct {
	Fetcher      datasources.ArticleFetcher
	SetRatingCmd command.Command[command.SetArticleRatingRequest, command.Empty]
	RatingType   domain.UserRatingType
}

func (c ArticleRatingSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", articleID))

	paramName := string(c.RatingType)

	var ratingValue bool
	switch vars[paramName] {
	case boolTrue:
		ratingValue = true
	case boolFalse:
		ratingValue = false
	default:
		logger.ErrorContext(ctx, "invalid "+paramName+" value", "value", vars[paramName])
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
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	req := command.SetArticleRatingRequest{
		UserID:        userID,
		ArticleHashID: articleID,
	}
	if c.RatingType == domain.RatingTypeThumbsUp {
		req.ThumbsUp = ratingValue
	} else {
		req.ThumbsDown = ratingValue
	}

	if _, err := c.SetRatingCmd.Execute(ctx, req); err != nil {
		logger.ErrorContext(ctx, "failed to set article rating", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
