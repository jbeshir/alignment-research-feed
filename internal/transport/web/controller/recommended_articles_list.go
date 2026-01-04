package controller

import (
	"encoding/json"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

type RecommendedArticlesList struct {
	Command *command.RecommendArticles
}

func (c RecommendedArticlesList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	userID := domain.UserIDFromContext(ctx)
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	articles, err := c.Command.Execute(ctx, userID, 10)
	if err != nil {
		logger.ErrorContext(ctx, "unable to get recommended articles", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if articles == nil {
		articles = []domain.Article{}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(ArticlesListResponse{
		Data:     articles,
		Metadata: ArticlesListMetadata{},
	}); err != nil {
		logger.ErrorContext(ctx, "unable to write recommended articles to response", "error", err)
	}
}
