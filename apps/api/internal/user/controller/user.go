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

func (u UserController) GetByID(c fiber.Ctx) error {
	return httpx.DoGetByID(c, u.users.GetByID)
}

func (u UserController) Search(c fiber.Ctx) error {
	return httpx.DoSearch(c, u.users.Search)
}

func (u UserController) Create(c fiber.Ctx) error {
	return httpx.DoCreate(c, u.users.Create)
}

func (u UserController) Update(c fiber.Ctx) error {
	return httpx.DoUpdate(c, u.users.Update)
}

func (u UserController) Delete(c fiber.Ctx) error {
	return httpx.DoDelete(c, u.users.Delete)
}
