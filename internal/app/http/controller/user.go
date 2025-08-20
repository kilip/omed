package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/contracts"
)

type UserController struct {
	users *contracts.UserService
}

func NewUserController(app *fiber.App, users *contracts.UserService) *UserController {
	return &UserController{
		users,
	}
}

func (ctl *UserController) Create() {

}
