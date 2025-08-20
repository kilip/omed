package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/domain/user"
)

type UserController struct {
	users *user.UserService
}

func NewUserController(users *user.UserService) *UserController {
	return &UserController{
		users,
	}
}

func (ctl UserController) LoadRoutes(app *fiber.App){

}

func (ctl *UserController) Create() {

}
