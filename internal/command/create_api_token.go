package command

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
)

// MaxAPITokensPerUser is the maximum number of active tokens a user can have.
const MaxAPITokensPerUser = 10

// ErrTokenLimitExceeded is returned when a user has reached the maximum number of active tokens.
var ErrTokenLimitExceeded = errors.New("user has reached maximum number of active tokens")

// APITokenPrefix is the prefix for API tokens in the Authorization header.
const APITokenPrefix = "user_api|"

// CreateAPITokenRequest is the request for the CreateAPIToken command.
type CreateAPITokenRequest struct {
	UserID string
	Name   *string
}

// CreateAPITokenResponse is the response from the CreateAPIToken command.
type CreateAPITokenResponse struct {
	TokenID   string
	FullToken string
	Prefix    string
}

// CreateAPIToken handles creating new API tokens.
type CreateAPIToken struct {
	TokenCounter datasources.UserAPITokenCounter
	TokenCreator datasources.APITokenCreator
}

// NewCreateAPIToken creates a properly initialized CreateAPIToken command.
func NewCreateAPIToken(
	tokenCounter datasources.UserAPITokenCounter,
	tokenCreator datasources.APITokenCreator,
) *CreateAPIToken {
	return &CreateAPIToken{
		TokenCounter: tokenCounter,
		TokenCreator: tokenCreator,
	}
}

// Execute creates a new API token for the user.
func (c *CreateAPIToken) Execute(ctx context.Context, req CreateAPITokenRequest) (CreateAPITokenResponse, error) {
	// Check token limit
	count, err := c.TokenCounter.CountUserActiveAPITokens(ctx, req.UserID)
	if err != nil {
		return CreateAPITokenResponse{}, fmt.Errorf("counting user tokens: %w", err)
	}

	if count >= MaxAPITokensPerUser {
		return CreateAPITokenResponse{}, ErrTokenLimitExceeded
	}

	// Generate cryptographically secure random token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return CreateAPITokenResponse{}, fmt.Errorf("generating random token: %w", err)
	}

	tokenHex := hex.EncodeToString(tokenBytes)
	fullToken := APITokenPrefix + tokenHex

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(fullToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Extract prefix (first 8 chars of the random portion)
	tokenPrefix := tokenHex[:8]

	// Generate UUID for token ID
	tokenID := uuid.New().String()

	// Store in database
	if err := c.TokenCreator.CreateAPIToken(ctx, tokenID, req.UserID, tokenHash, tokenPrefix, req.Name, nil); err != nil {
		return CreateAPITokenResponse{}, fmt.Errorf("creating token: %w", err)
	}

	return CreateAPITokenResponse{
		TokenID:   tokenID,
		FullToken: fullToken,
		Prefix:    tokenPrefix,
	}, nil
}
