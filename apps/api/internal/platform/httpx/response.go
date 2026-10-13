package httpx

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// NewMeta constructs a response Meta object populated with the request ID, current timestamp, and cursor.
func NewMeta(c fiber.Ctx, cursor string) Meta {
	return Meta{
		RequestID: requestid.FromContext(c),
		Timestamp: time.Now(),
		Cursor:    cursor,
	}
}

// Success serializes data and metadata into the standard WebResponse envelope with the given HTTP status.
func Success[T any](c fiber.Ctx, status int, data T) error {
	return c.Status(status).JSON(WebResponse[T]{
		Data: data,
		Meta: NewMeta(c, ""),
	})
}

// WithCursor serializes data and pagination cursor into a 200 OK WebResponse envelope.
func WithCursor[T any](c fiber.Ctx, data T, cursor string) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse[T]{
		Data: data,
		Meta: NewMeta(c, cursor),
	})
}

// OK serializes data into a 200 OK WebResponse envelope.
func OK[T any](c fiber.Ctx, data T) error {
	return Success(c, fiber.StatusOK, data)
}

// Created serializes data into a 201 Created WebResponse envelope.
func Created[T any](c fiber.Ctx, data T) error {
	return Success(c, fiber.StatusCreated, data)
}
