package router

import (
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

func requireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := domain.UserIDFromContext(r.Context())
		if userID == "" {
			logger := domain.LoggerFromContext(r.Context())
			logger.ErrorContext(r.Context(), "attempt to use endpoint requiring auth without user ID")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireNonAPITokenAuthMiddleware requires authentication but disallows API token auth.
// This is used for token management endpoints to prevent bootstrapping issues
// (can't use API tokens to create more API tokens).
func requireNonAPITokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is authenticated
		userID := domain.UserIDFromContext(r.Context())
		if userID == "" {
			logger := domain.LoggerFromContext(r.Context())
			logger.ErrorContext(r.Context(), "attempt to use protected endpoint without authentication")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify this was not API token auth
		authMethod := domain.AuthMethodFromContext(r.Context())
		if authMethod == domain.AuthMethodAPIToken {
			logger := domain.LoggerFromContext(r.Context())
			logger.WarnContext(r.Context(), "attempt to use non-API-token endpoint with API token")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"This endpoint cannot be accessed with an API token."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
