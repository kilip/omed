package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/domain/user"
)

type UserController struct {
	users user.UserService
}

func NewUserController(app *fiber.App, users user.UserService) *UserController {
	ctl := &UserController{
		users,
	}
	ctl.loadRoutes(app)
	return ctl
}

func (ctl UserController) loadRoutes(app *fiber.App){
	app.Get("/", ctl.Get)
}

func (ctl *UserController) Get(ctx *fiber.Ctx) error {
	ctx.Status(fiber.StatusOK).JSON(&fiber.Map{
		"message": "ok",
	})

	return nil
}
