// Package domain defines core domain entities and business errors for the core module.
package domain

import (
	"time"
)

// AccessTokenStatus represents the status of a personal access token.
type AccessTokenStatus string

const (
	// AccessTokenActive indicates that the token is currently valid and usable.
	AccessTokenActive AccessTokenStatus = "active"
	// AccessTokenExpired indicates that the token has expired.
	AccessTokenExpired AccessTokenStatus = "expired"
	// AccessTokenRevoked indicates that the token has been explicitly revoked.
	AccessTokenRevoked AccessTokenStatus = "revoked"
)

// AccessToken represents a personal access token for API and service authentication.
type AccessToken struct {
	ID        int64
	UserID    int64
	Name      string
	TokenHash string
	Prefix    string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// Status returns the token status relative to now. Revoked status wins over expired.
func (t AccessToken) Status(now time.Time) AccessTokenStatus {
	if t.RevokedAt != nil {
		return AccessTokenRevoked
	}
	if !t.ExpiresAt.IsZero() && !now.Before(t.ExpiresAt) {
		return AccessTokenExpired
	}
	return AccessTokenActive
}
