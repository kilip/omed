package model

import "github.com/kilip/omed/cms/internal/entity"

type LoginRequest struct {
	Email string
	Password string
}

type LoginResponse struct {
	UserID uint64 `json:"userId"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type VerifyUserRequest struct {
	Token string
}

type AuthResponse struct {
	UserID uint64 `json:"userId"`
	Name string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type UserProfileRequest struct {
	
}

func CreateLoginResponse(user *entity.User, token string) *LoginResponse {
	return &LoginResponse{
		UserID: user.ID,
		Email: user.Email,
		Token: token,
	}
}
