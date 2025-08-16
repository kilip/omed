package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/kilip/omed/cms/internal/service"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	Users *service.UserService
	Log *logrus.Logger
}

func NewUserController(users *service.UserService, log *logrus.Logger) *UserController{
	return &UserController{
		Users: users,
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

	response, err := c.Users.Register(ctx.UserContext(), request)
	if err != nil {
		c.Log.Warnf("Failed to register user : %+v", err)
		return err
	}
	
	return ctx.Status(201).JSON(model.Resource[*model.UserResponse]{Data: response})
}


func (c UserController) Update(ctx *fiber.Ctx) error{
	return ctx.Status(200).SendString("OK")
}

