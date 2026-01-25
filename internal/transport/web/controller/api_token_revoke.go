package controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// APITokenRevoke handles DELETE /v1/tokens/{token_id} to revoke a token.
type APITokenRevoke struct {
	TokenRevoker datasources.APITokenRevoker
}

func (c APITokenRevoke) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := domain.LoggerFromContext(ctx)

	userID := domain.UserIDFromContext(ctx)
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	tokenID := vars["token_id"]
	if tokenID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := c.TokenRevoker.RevokeAPIToken(ctx, tokenID, userID); err != nil {
		logger.ErrorContext(ctx, "unable to revoke API token", "error", err, "token_id", tokenID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
