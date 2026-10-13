package model

import "time"

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateUserRequest struct {
	Name   string `json:"name" validate:"required"`
	Email  string `json:"email" validate:"required,email"`
	Avatar string `json:"avatar,omitempty" validate:"omitempty,url"`
}

type UpdateUserRequest struct {
	Name   string `json:"name,omitempty" validate:"required"`
	Avatar string `json:"avatar,omitempty" validate:"omitempty,url"`
}

type SearchRequest struct {
	Name   string `json:"name,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}
