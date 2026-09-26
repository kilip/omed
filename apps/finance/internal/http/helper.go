package http

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/finance/internal/model"
)

func GetUser(c fiber.Ctx) model.AuthenticatedUser {
	authenticated := fiber.Locals[model.AuthenticatedUser](c, "user")

	return authenticated
}

func buildMeta(c fiber.Ctx, cursor *model.Cursor) model.Meta {
	return model.Meta{
		RequestID: requestid.FromContext(c),
		Timestamp: time.Now(),
		Cursor:    cursor,
	}
}

func OK[T any](c fiber.Ctx, data T) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse[T]{
		Data: data,
		Meta: buildMeta(c, nil),
	})
}
