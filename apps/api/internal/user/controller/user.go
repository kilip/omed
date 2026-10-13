package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/internal/user/model"
)

//go:generate go run go.uber.org/mock/mockgen -source=user.go -destination=mock/mock_user.go -package=mock
type UserService interface {
	GetByID(c context.Context, id int64) (model.UserResponse, error)
	Search(c context.Context, r model.SearchRequest) ([]model.UserResponse, string, error)
	Create(c context.Context, r model.CreateUserRequest) (model.UserResponse, error)
	Update(c context.Context, id int64, r model.UpdateUserRequest) (model.UserResponse, error)
	Delete(c context.Context, id int64) error
}

type UserController struct {
	users UserService
}

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
	return httpx.DoGetByID(c, u.users.GetByID)
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
	return httpx.DoSearch(c, u.users.Search)
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
	return httpx.DoCreate(c, u.users.Create)
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
	return httpx.DoUpdate(c, u.users.Update)
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
