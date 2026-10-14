// Package domain defines core domain entities and business errors for the core module.
package domain

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

const (
	// ProviderGoogle identifies Google as an OAuth2 provider.
	ProviderGoogle = "google"
	// ProviderGithub identifies GitHub as an OAuth2 provider.
	ProviderGithub = "github"
)

var (
	// ErrInvalidProvider indicates that the requested OAuth provider is not supported.
	ErrInvalidProvider = errors.New("OAUTH_INVALID_PROVIDER")
	// ErrInvalidState indicates that the OAuth state parameter is missing, expired, or invalid.
	ErrInvalidState = errors.New("OAUTH_INVALID_STATE")
	// ErrOAuthExchangeFailed indicates failure during exchanging authorization code for token.
	ErrOAuthExchangeFailed = errors.New("OAUTH_EXCHANGE_FAILED")
	// ErrFetchUserInfoFailed indicates failure during retrieving user profile from provider.
	ErrFetchUserInfoFailed = errors.New("OAUTH_USERINFO_FAILED")
	// ErrSessionNotFound indicates that the requested session does not exist.
	ErrSessionNotFound = errors.New("SESSION_NOT_FOUND")
	// ErrSessionExpired indicates that the session has expired.
	ErrSessionExpired = errors.New("SESSION_EXPIRED")
)

func init() {
	base.AddToErrorMap(ErrInvalidProvider, fiber.StatusBadRequest, "unsupported oauth provider")
	base.AddToErrorMap(ErrInvalidState, fiber.StatusBadRequest, "invalid or expired oauth state")
	base.AddToErrorMap(ErrOAuthExchangeFailed, fiber.StatusBadRequest, "failed to exchange oauth code")
	base.AddToErrorMap(ErrFetchUserInfoFailed, fiber.StatusBadGateway, "failed to fetch user info from provider")
	base.AddToErrorMap(ErrSessionNotFound, fiber.StatusUnauthorized, "session not found")
	base.AddToErrorMap(ErrSessionExpired, fiber.StatusUnauthorized, "session expired")
}

// Account represents the domain model for an external OAuth account linked to a user.
type Account struct {
	ID                    int64
	UserID                int64
	ProviderID            string
	AccountID             string
	AccessToken           string
	RefreshToken          string
	IDToken               string
	AccessTokenExpiresAt  *time.Time
	RefreshTokenExpiresAt *time.Time
	Scope                 string
	Password              string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Session represents the domain model for an active user session.
type Session struct {
	ID        int64
	Token     string
	UserID    int64
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	UpdatedAt time.Time
}

