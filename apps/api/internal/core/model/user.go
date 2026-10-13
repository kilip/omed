// Package model provides Data Transfer Objects (DTOs) and request/response payloads for user endpoints.
package model

import "time"

// UserResponse represents the public user profile payload returned by user API endpoints.
type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateUserRequest defines the incoming payload for registering a new user.
type CreateUserRequest struct {
	Name   string `json:"name" validate:"required"`
	Email  string `json:"email" validate:"required,email"`
	Avatar string `json:"avatar,omitempty" validate:"omitempty,url"`
}

// UpdateUserRequest defines the incoming payload for updating an existing user's profile.
type UpdateUserRequest struct {
	Name   string `json:"name,omitempty" validate:"required"`
	Avatar string `json:"avatar,omitempty" validate:"omitempty,url"`
}

// SearchRequest defines filter criteria and cursor pagination options for searching users.
type SearchRequest struct {
	Name   string `json:"name,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}
