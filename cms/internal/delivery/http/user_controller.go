package http

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/kilip/omed/cms/internal/service"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	Service *service.UserService
	Log *logrus.Logger
}

func NewUserController(service *service.UserService, log *logrus.Logger) *UserController{
	return &UserController{
		Service: service,
		Log: log,
	}
}

func (c UserController) Register(ctx *fiber.Ctx) error {
	request := new(model.RegisterUserRequest)
	err := ctx.BodyParser(request)
	if err != nil {
		c.Log.Warnf("Failed to parse request body: %+v", err)
		return fiber.ErrBadRequest
	}

	response, err := c.Service.Register(ctx.UserContext(), request)
	if err != nil {
		c.Log.Warnf("Failed to register user : %+v", err)
		return err
	}
	fmt.Printf("Data: %+v \n", response)
	return ctx.Status(201).JSON(model.Resource[*model.UserResponse]{Data: response})
}