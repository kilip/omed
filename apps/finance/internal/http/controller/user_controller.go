package controller

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
)

type PingResponse struct {
	User model.AuthenticatedUser `json:"user"`
}

type UserController struct {
}

func NewUserController(app *fiber.App) UserController {
	ctl := UserController{}

	ctl.initRoutes(app)
	return ctl
}

func (s UserController) initRoutes(app *fiber.App) {
	app.Get("/ping", s.ping)
}

func (s UserController) ping(c fiber.Ctx) error {
	return http.OK(c, PingResponse{User: http.GetUser(c)})
}
