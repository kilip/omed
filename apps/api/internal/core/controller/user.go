// Package controller provides HTTP handlers and endpoint bindings for user and workspace management.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// UserService defines the business logic operations required by UserController.
type UserService interface {
	GetByID(c context.Context, id int64) (domain.User, error)
	Update(c context.Context, id int64, r model.UpdateUserRequest) (domain.User, error)
}

// UserController handles HTTP requests for user profile and preferences.
type UserController struct {
	users UserService
}

// NewUserController constructs a new UserController backed by the specified UserService.
func NewUserController(users UserService) UserController {
	return UserController{users: users}
}

// GetMe retrieves the profile of the current authenticated user.
//
//	@Summary		Get current user profile
//	@Description	Retrieve authenticated user's profile and preferences (locale, currency)
//	@Tags			Users
//	@Produce		json
//	@Success		200	{object}	httpx.WebResponse[model.UserResponse]
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me [get]
func (ctl UserController) GetMe(c fiber.Ctx) error {
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}

	usr, err := ctl.users.GetByID(c, user.ID)
	if err != nil {
		return err
	}

	return httpx.OK(c, toUserResponse(usr))
}

// UpdateMe updates the profile and preferences of the current authenticated user.
//
//	@Summary		Update current user profile
//	@Description	Update authenticated user's profile and preferences (locale, currency, name, avatar)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.UpdateUserRequest	true	"Update user payload"
//	@Success		200		{object}	httpx.WebResponse[model.UserResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me [put]
func (ctl UserController) UpdateMe(c fiber.Ctx) error {
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}

	var req model.UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	usr, err := ctl.users.Update(c, user.ID, req)
	if err != nil {
		return err
	}

	return httpx.OK(c, toUserResponse(usr))
}

func toUserResponse(u domain.User) model.UserResponse {
	return model.UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Avatar:    u.Avatar,
		Locale:    u.Locale,
		Currency:  u.Currency,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
