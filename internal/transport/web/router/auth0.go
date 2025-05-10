package router

import (
	"fmt"
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const auth0AuthHeaderPrefix = "Bearer auth0|"

func SetupAuth0Middleware(auth0Domain, auth0Audience string) (func(http.Handler) http.Handler, error) {
	issuerURL, err := url.Parse("https://" + auth0Domain + "/")
	if err != nil {
		return nil, fmt.Errorf("failed to parse the issuer url: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)
	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{auth0Audience},
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT validator: %w", err)
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		logger := domain.LoggerFromContext(r.Context())
		logger.WarnContext(r.Context(), "encountered error while validating JWT", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Failed to validate JWT."}`))
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	return func(next http.Handler) http.Handler {
		mwHandler := middleware.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
			ctx := domain.ContextWithUserID(r.Context(), token.RegisteredClaims.Subject)

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Some of our endpoints add extra data when authenticated,
			// but are also allowed when unauthenticated.
			// To allow this, we validate and attach user data only when an auth header is set in this MW,
			// passing requests without an auth header through unchanged.
			// Separate MW on the logged-in only endpoints asserts that it exists.
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, auth0AuthHeaderPrefix) {
				r.Header.Set("Authorization", "Bearer "+authHeader[len(auth0AuthHeaderPrefix):])
				mwHandler.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}
