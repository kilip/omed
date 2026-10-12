package httpx

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func NewMeta(c fiber.Ctx, cursor string) Meta {
	return Meta{
		RequestID: requestid.FromContext(c),
		Timestamp: time.Now(),
		Cursor:    cursor,
	}
}

func Success[T any](c fiber.Ctx, status int, data T) error {
	return c.Status(status).JSON(WebResponse[T]{
		Data: data,
		Meta: NewMeta(c, ""),
	})
}

func WithCursor[T any](c fiber.Ctx, data T, cursor string) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse[T]{
		Data: data,
		Meta: NewMeta(c, cursor),
	})
}

func OK[T any](c fiber.Ctx, data T) error {
	return Success(c, fiber.StatusOK, data)
}

func Created[T any](c fiber.Ctx, data T) error {
	return Success(c, fiber.StatusCreated, data)
}
