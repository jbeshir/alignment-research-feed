package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// AuthResult represents the result of a successful authentication.
type AuthResult struct {
	UserID string
	Method domain.AuthMethod
}

// AuthValidator attempts to validate authentication from a request.
// Returns nil, nil if this validator doesn't apply (wrong auth type).
// Returns AuthResult, nil on success.
// Returns nil, error if validation was attempted but failed.
type AuthValidator func(r *http.Request) (*AuthResult, error)

// NewAuthMiddleware creates a middleware that validates requests using multiple authentication methods.
func NewAuthMiddleware(validators []AuthValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, validate := range validators {
				result, err := validate(r)
				if result == nil && err == nil {
					continue // This validator doesn't apply
				}

				if err != nil {
					logger := domain.LoggerFromContext(r.Context())
					logger.WarnContext(r.Context(), "authentication failed", "error", err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = fmt.Fprintf(w, `{"message":"%s"}`, err.Error())
					return
				}

				ctx := domain.ContextWithUserID(r.Context(), result.UserID)
				ctx = domain.ContextWithAuthMethod(ctx, result.Method)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No validator matched - continue without auth (for public endpoints)
			next.ServeHTTP(w, r)
		})
	}
}

// NewAuth0Validator creates a validator for Auth0 JWT tokens.
func NewAuth0Validator(auth0Domain, auth0Audience string) (AuthValidator, error) {
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

	return func(r *http.Request) (*AuthResult, error) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer auth0|") {
			return nil, nil
		}

		token, err := jwtValidator.ValidateToken(r.Context(), authHeader[len("Bearer auth0|"):])
		if err != nil {
			return nil, fmt.Errorf("invalid JWT token")
		}

		claims := token.(*validator.ValidatedClaims)
		return &AuthResult{
			UserID: claims.RegisteredClaims.Subject,
			Method: domain.AuthMethodAuth0,
		}, nil
	}, nil
}

// NewAPITokenValidator creates a validator for API tokens.
// It asynchronously updates the token's last_used_at timestamp on successful validation.
func NewAPITokenValidator(
	ctx context.Context,
	tokenGetter datasources.APITokenByHashGetter,
	lastUsedUpdater datasources.APITokenLastUsedUpdater,
) AuthValidator {
	// Asynchronous best-effort tracking of the last used time of each token.
	// If the service restarts up to the buffer size of updates here might be lost, but this is tolerable.
	// We apply backpressure at the point the channel becomes full.
	updateChan := make(chan string, 100)
	go func() {
		for tokenID := range updateChan {
			updateErr := lastUsedUpdater.UpdateAPITokenLastUsed(context.WithoutCancel(ctx), tokenID)
			if updateErr != nil {
				logger := domain.LoggerFromContext(ctx).With("token", tokenID)
				logger.WarnContext(context.WithoutCancel(ctx),
					"failed to update last used time for token",
					"error", updateErr)
			}
		}
	}()

	return func(r *http.Request) (*AuthResult, error) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer "+command.APITokenPrefix) {
			return nil, nil
		}

		fullToken := authHeader[len("Bearer "):]
		hash := sha256.Sum256([]byte(fullToken))
		tokenHash := hex.EncodeToString(hash[:])

		token, err := tokenGetter.GetAPITokenByHash(r.Context(), tokenHash)
		if err != nil {
			return nil, fmt.Errorf("invalid API token")
		}

		if !token.IsActive() {
			return nil, fmt.Errorf("API token is revoked or expired")
		}

		select {
		case updateChan <- token.ID:
		default:
		}

		return &AuthResult{
			UserID: token.UserID,
			Method: domain.AuthMethodAPIToken,
		}, nil
	}
}
