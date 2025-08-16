package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/delivery/http/middleware"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/kilip/omed/cms/internal/service"
	"github.com/sirupsen/logrus"
)

type AuthController struct {
	Auth *service.AuthService
	Log *logrus.Logger
}

func NewAuthController(auth *service.AuthService, log *logrus.Logger) *AuthController{
	return &AuthController{
		Auth: auth,
		Log: log,
	}
}

func (c AuthController) Login(ctx *fiber.Ctx) error {
	request := new(model.LoginRequest)
	if err := ctx.BodyParser(request); err != nil {
		c.Log.Warnf("Failed to parse request body : %+v", err)
		return fiber.ErrBadRequest
	}

	response, err := c.Auth.Login(ctx.UserContext(), request)
	if err != nil {
		c.Log.Warnf("Failed to login user : %+v", err)
		return err
	}

	return ctx.JSON(model.Resource[*model.LoginResponse]{Data: response})
}

func (c AuthController) Profile(ctx *fiber.Ctx) error {
	auth := middleware.GetUser(ctx)
	return ctx.JSON(model.Resource[*model.AuthResponse]{Data: auth})
}
