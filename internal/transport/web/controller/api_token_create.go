package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// APITokenCreateRequest is the JSON request body for creating a token.
type APITokenCreateRequest struct {
	Name string `json:"name,omitempty"`
}

// APITokenCreateResponse is the JSON response for a created token.
type APITokenCreateResponse struct {
	ID     string `json:"id"`
	Token  string `json:"token"`
	Prefix string `json:"prefix"`
}

// APITokenCreate handles POST /v1/tokens to create a new API token.
type APITokenCreate struct {
	CreateCmd *command.CreateAPIToken
}

func (c APITokenCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	userID := domain.UserIDFromContext(ctx)
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var reqBody APITokenCreateRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			logger.ErrorContext(ctx, "unable to parse request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	req := command.CreateAPITokenRequest{
		UserID: userID,
	}
	if reqBody.Name != "" {
		req.Name = &reqBody.Name
	}

	result, err := c.CreateCmd.Execute(ctx, req)
	if err != nil {
		logger.ErrorContext(ctx, "unable to create API token", "error", err)
		if errors.Is(err, command.ErrTokenLimitExceeded) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			if encErr := json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			}); encErr != nil {
				logger.ErrorContext(ctx, "unable to write error response", "error", encErr)
			}
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(APITokenCreateResponse{
		ID:     result.TokenID,
		Token:  result.FullToken,
		Prefix: result.Prefix,
	}); err != nil {
		logger.ErrorContext(ctx, "unable to write response", "error", err)
	}
}
