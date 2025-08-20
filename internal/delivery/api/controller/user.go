package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/delivery/api"
	"github.com/kilip/omed/internal/domain/user"
	"github.com/kilip/omed/internal/infra/database/repository"
	"github.com/kilip/omed/internal/service"
)

type UserController struct {
	users user.UserService
	server *api.Server
}

func NewUserController(server *api.Server) *UserController {
	repo := repository.NewUserRepository(server.Query)
	service := service.NewUserService(repo)
	ctl := &UserController{
		service,
		server,
	}
	ctl.loadRoutes()
	return ctl
}

func (ctl UserController) loadRoutes(){
	app := ctl.server.App

	app.Post("/user/register", ctl.Register)
}

func (ctl *UserController) Register(ctx *fiber.Ctx) error {
	ctx.Status(fiber.StatusOK).JSON(&fiber.Map{
		"message": "ok",
	})

	return nil
}
