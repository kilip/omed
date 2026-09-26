package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http"
)

func GetFiber(cfg Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "finance",
		ErrorHandler: NewErrorHandler(),
	})

	return app
}

func NewErrorHandler() fiber.ErrorHandler {
	return func(ctx fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		if errors.Is(err, http.ErrInvalidToken) {
			code = fiber.StatusUnauthorized
		}

		return ctx.Status(code).JSON(fiber.Map{
			"errors": err.Error(),
		})
	}
}
