// Package model provides Data Transfer Objects (DTOs) and request/response payloads for user endpoints.
package model

import "time"

// UserResponse represents the public user profile payload returned by user API endpoints.
type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar,omitempty"`
	Locale    string    `json:"locale"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateUserRequest defines the incoming payload for registering a new user.
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Avatar   string `json:"avatar,omitempty" validate:"omitempty,url"`
	Locale   string `json:"locale,omitempty" validate:"omitempty,oneof=en id"`
	Currency string `json:"currency,omitempty" validate:"omitempty,len=3"`
}

// UpdateUserRequest defines the incoming payload for updating an existing user's profile.
type UpdateUserRequest struct {
	Name     string `json:"name,omitempty" validate:"omitempty"`
	Avatar   string `json:"avatar,omitempty" validate:"omitempty,url"`
	Locale   string `json:"locale,omitempty" validate:"omitempty,oneof=en id"`
	Currency string `json:"currency,omitempty" validate:"omitempty,len=3"`
}

// SearchRequest defines filter criteria and cursor pagination options for searching users.
type SearchRequest struct {
	Name   string `json:"name,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}
