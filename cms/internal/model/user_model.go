package model

import (
	"github.com/kilip/omed/cms/internal/entity"
)

type RegisterUserRequest struct {
	PlainPassword string `json:"plainPassword" validate:"required,max=100"`
	Name string `json:"name" validate:"required,max=100"`
	Email string `json:"email" validate:"required,email"`
}

type UserResponse struct {
	ID uint64 `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	CreatedAt int64 `json:"createdAt,omitempty"`
	UpdatedAt int64 `json:"updatedAt,omitempty"`
	Token string `json:"token,omitempty"`
}

func UserToResponse(user *entity.User) *UserResponse{
	return &UserResponse{
		Name: user.Name,
		Email: user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
