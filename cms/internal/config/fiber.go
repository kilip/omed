package config

import "github.com/gofiber/fiber/v2"

func NewErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		return ctx.Status(code).JSON(fiber.Map{
			"errors": err.Error(),
		})
	}
}

func NewFiber(config *OmedConfig) *fiber.App{
	var app = fiber.New(fiber.Config{
		AppName: config.AppName,
		ErrorHandler: NewErrorHandler(),
		Prefork: config.Web.Prefork,
	})

	return app
}
