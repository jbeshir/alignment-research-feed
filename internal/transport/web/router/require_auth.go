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
