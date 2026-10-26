// Package model defines data transfer objects (DTOs) and request/response payloads for the core module.
package model

import "time"

// SignInSocialRequest represents the payload for initiating a social OAuth flow.
type SignInSocialRequest struct {
	Provider           string `json:"provider" validate:"required"`
	CallbackURL        string `json:"callbackURL,omitempty"`
	ErrorCallbackURL   string `json:"errorCallbackURL,omitempty"`
	NewUserCallbackURL string `json:"newUserCallbackURL,omitempty"`
	Locale             string `json:"locale,omitempty"`
}

// SignInSocialResponse contains the redirection URL returned by social sign-in.
type SignInSocialResponse struct {
	URL      string `json:"url"`
	Redirect bool   `json:"redirect"`
	Nonce    string `json:"-"`
}

// SessionData holds the serialized session information.
type SessionData struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserData holds user profile information returned in session responses.
type UserData struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Avatar        string `json:"image,omitempty"`
	Locale        string `json:"locale"`
	Currency      string `json:"currency"`
	EmailVerified bool   `json:"emailVerified"`
}

// SessionResponse represents the response containing both session and user details.
type SessionResponse struct {
	Session *SessionData `json:"session"`
	User    *UserData    `json:"user"`
}

// OAuthUserInfo represents normalized user information fetched from an OAuth2 provider.
type OAuthUserInfo struct {
	ID            string
	Email         string
	Name          string
	Avatar        string
	EmailVerified bool
	Locale        string
}

// OAuthStateClaims contains data encoded and signed within the OAuth state parameter.
type OAuthStateClaims struct {
	Nonce            string `json:"nonce"`
	Provider         string `json:"provider"`
	CallbackURL      string `json:"callbackURL,omitempty"`
	ErrorCallbackURL string `json:"errorCallbackURL,omitempty"`
	Locale           string `json:"locale,omitempty"`
	ExpiresAt        int64  `json:"exp"`
}
