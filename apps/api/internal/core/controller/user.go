// Package controller provides HTTP handlers and endpoint bindings for user management.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// UserService defines the business logic operations required by UserController.
type UserService interface {
	GetByID(c context.Context, id int64) (domain.User, error)
	Search(c context.Context, r model.SearchRequest) ([]domain.User, string, error)
	Create(c context.Context, r model.CreateUserRequest) (domain.User, error)
	Update(c context.Context, id int64, r model.UpdateUserRequest) (domain.User, error)
	Delete(c context.Context, id int64) error
}

// UserController handles HTTP requests for user CRUD operations and search.
type UserController struct {
	users UserService
}

// NewUserController constructs a new UserController backed by the specified UserService.
func NewUserController(users UserService) UserController {
	return UserController{users}
}

// GetByID gets a user by ID.
//
//	@Summary		Get user by ID
//	@Description	Retrieve user detail by ID
//	@Tags			Users
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	httpx.WebResponse[model.UserResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{id} [get]
func (u UserController) GetByID(c fiber.Ctx) error {
	return httpx.DoGetByID(c, u.users.GetByID, toUserResponse)
}

// Search searches users based on query parameters.
//
//	@Summary		Search users
//	@Description	Search users matching filter criteria with pagination cursor
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.SearchRequest	true	"Search filter criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.UserResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/search [post]
func (u UserController) Search(c fiber.Ctx) error {
	return httpx.DoSearch(c, u.users.Search, toUserResponse)
}

// Create creates a new user.
//
//	@Summary		Create user
//	@Description	Create a new user
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.CreateUserRequest	true	"User creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.UserResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users [post]
func (u UserController) Create(c fiber.Ctx) error {
	return httpx.DoCreate(c, u.users.Create, toUserResponse)
}

// Update updates an existing user by ID.
//
//	@Summary		Update user
//	@Description	Update an existing user by ID
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"User ID"
//	@Param			request	body		model.UpdateUserRequest	true	"User update payload"
//	@Success		200		{object}	httpx.WebResponse[model.UserResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{id} [put]
func (u UserController) Update(c fiber.Ctx) error {
	return httpx.DoUpdate(c, u.users.Update, toUserResponse)
}

// Delete deletes a user by ID.
//
//	@Summary		Delete user
//	@Description	Delete a user by ID
//	@Tags			Users
//	@Produce		json
//	@Param			id	path	int	true	"User ID"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{id} [delete]
func (u UserController) Delete(c fiber.Ctx) error {
	return httpx.DoDelete(c, u.users.Delete)
}

func toUserResponse(user domain.User) model.UserResponse {
	return model.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
