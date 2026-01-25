package datasources

import (
	"context"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// APITokenCreator creates a new API token.
type APITokenCreator interface {
	CreateAPIToken(
		ctx context.Context,
		id, userID, tokenHash, tokenPrefix string,
		name *string,
		expiresAt *time.Time,
	) error
}

// APITokenByHashGetter retrieves an API token by its hash.
type APITokenByHashGetter interface {
	GetAPITokenByHash(ctx context.Context, tokenHash string) (domain.APIToken, error)
}

// APITokenLastUsedUpdater updates the last_used_at timestamp for a token.
type APITokenLastUsedUpdater interface {
	UpdateAPITokenLastUsed(ctx context.Context, tokenID string) error
}

// UserAPITokenLister lists all tokens for a user.
type UserAPITokenLister interface {
	ListUserAPITokens(ctx context.Context, userID string) ([]domain.APIToken, error)
}

// UserAPITokenCounter counts active tokens for a user.
type UserAPITokenCounter interface {
	CountUserActiveAPITokens(ctx context.Context, userID string) (int64, error)
}

// APITokenRevoker revokes a token.
type APITokenRevoker interface {
	RevokeAPIToken(ctx context.Context, tokenID, userID string) error
}

// APITokenRepository combines all API token operations.
type APITokenRepository interface {
	APITokenCreator
	APITokenByHashGetter
	APITokenLastUsedUpdater
	UserAPITokenLister
	UserAPITokenCounter
	APITokenRevoker
}
