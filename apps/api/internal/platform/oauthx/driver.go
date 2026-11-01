package oauthx

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
)

// ErrUnsupportedProvider is returned when no driver is registered for a provider ID.
var ErrUnsupportedProvider = errors.New("oauthx: unsupported provider")

// Profile is the normalized user profile returned by a provider.
type Profile struct {
	ID            string
	Email         string
	Name          string
	Avatar        string
	EmailVerified bool
	Locale        string
}

// Driver encapsulates all provider-specific OAuth2 behavior.
type Driver interface {
	// ID returns the provider identifier, e.g. "google".
	ID() string
	// Scopes returns the scopes requested from the provider.
	Scopes() []string
	// GetAuthURL builds the provider authorization URL.
	GetAuthURL(state, redirectURL string) string
	// ExchangeCode exchanges an authorization code for a token.
	ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error)
	// FetchUserProfile retrieves and normalizes the user profile.
	FetchUserProfile(ctx context.Context, token *oauth2.Token) (*Profile, error)
}
