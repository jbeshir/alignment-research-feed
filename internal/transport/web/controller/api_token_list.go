package controller

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// APITokenListItem represents a token in the list response.
type APITokenListItem struct {
	ID         string     `json:"id"`
	Prefix     string     `json:"prefix"`
	Name       *string    `json:"name,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Revoked    bool       `json:"revoked"`
}

// APITokenListResponse is the JSON response for listing tokens.
type APITokenListResponse struct {
	Data []APITokenListItem `json:"data"`
}

// APITokenList handles GET /v1/tokens to list user's API tokens.
type APITokenList struct {
	TokenLister datasources.UserAPITokenLister
}

func (c APITokenList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	userID := domain.UserIDFromContext(ctx)
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokens, err := c.TokenLister.ListUserAPITokens(ctx, userID)
	if err != nil {
		logger.ErrorContext(ctx, "unable to list API tokens", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	items := make([]APITokenListItem, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, APITokenListItem{
			ID:         token.ID,
			Prefix:     token.Prefix,
			Name:       token.Name,
			CreatedAt:  token.CreatedAt,
			LastUsedAt: token.LastUsedAt,
			ExpiresAt:  token.ExpiresAt,
			Revoked:    token.RevokedAt != nil,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(APITokenListResponse{
		Data: items,
	}); err != nil {
		logger.ErrorContext(ctx, "unable to write response", "error", err)
	}
}
