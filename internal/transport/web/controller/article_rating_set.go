package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type RatingType int

const (
	RatingTypeThumbsUp RatingType = iota
	RatingTypeThumbsDown
)

type ArticleRatingSet struct {
	Fetcher      datasources.ArticleFetcher
	SetRatingCmd command.Command[command.SetArticleRatingRequest, command.Empty]
	RatingType   RatingType
}

func (c ArticleRatingSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID := vars["article_id"]
	logger := domain.LoggerFromContext(r.Context())
	ctx := domain.ContextWithLogger(r.Context(), logger.With("article_id", articleID))

	var varName string
	if c.RatingType == RatingTypeThumbsUp {
		varName = "thumbs_up"
	} else {
		varName = "thumbs_down"
	}

	var ratingValue bool
	switch vars[varName] {
	case boolTrue:
		ratingValue = true
	case boolFalse:
		ratingValue = false
	default:
		logger.ErrorContext(ctx, "invalid "+varName+" value", "value", vars[varName])
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
	req := command.SetArticleRatingRequest{
		UserID:        userID,
		ArticleHashID: articleID,
	}
	if c.RatingType == RatingTypeThumbsUp {
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
