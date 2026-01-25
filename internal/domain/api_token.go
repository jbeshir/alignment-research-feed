package domain

import "time"

// APIToken represents a user's API token for programmatic access.
type APIToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"-"`
	TokenHash  string     `json:"-"`
	Prefix     string     `json:"prefix"`
	Name       *string    `json:"name,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"-"`
}

// IsActive returns true if the token is not revoked and not expired.
func (t APIToken) IsActive() bool {
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return false
	}
	return true
}
