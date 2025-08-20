package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/domain/user"
)

type UserController struct {
	users *user.UserService
}

func NewUserController(app *fiber.App, users *user.UserService) *UserController {
	return &UserController{
		users,
	}
}

func (ctl *UserController) Create() {

}
