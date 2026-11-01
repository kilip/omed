// Package model provides Data Transfer Objects (DTOs) and request/response payloads for the core module.
package model

import "time"

// CreateAccessTokenRequest represents the request payload for creating a new personal access token.
type CreateAccessTokenRequest struct {
	Name          string `json:"name"`
	ExpiresInDays int    `json:"expiresInDays"`
}

// AccessTokenResponse represents a personal access token without secret material.
type AccessTokenResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreatedAccessTokenResponse represents a newly created personal access token including the plaintext secret.
type CreatedAccessTokenResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Status    string    `json:"status"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}
