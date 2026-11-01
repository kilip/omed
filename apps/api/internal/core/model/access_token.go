// Package model provides Data Transfer Objects (DTOs) and request/response payloads for the core module.
package model

// CreateAccessTokenRequest represents the request payload for creating a new personal access token.
type CreateAccessTokenRequest struct {
	Name          string `json:"name"`
	ExpiresInDays int    `json:"expiresInDays"`
}
