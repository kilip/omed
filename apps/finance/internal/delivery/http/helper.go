package http

import (
	"errors"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/authz"
	"github.com/kilip/omed/finance/internal/model"
)

func GetUser(c fiber.Ctx) model.AuthenticatedUser {
	authenticated := fiber.Locals[*model.AuthenticatedUser](c, "user")

	return *authenticated
}

func RequirePermission(enforcer *casbin.Enforcer, obj authz.Resource, act authz.Action) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := fiber.Locals[*model.AuthenticatedUser](c, "user")

		for _, role := range user.WorkspaceRoles {
			if ok, _ := enforcer.Enforce(role, user.WorkspaceID, string(obj), string(act)); ok {
				return c.Next()
			}

		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
}

func buildMeta(c fiber.Ctx, cursor *model.Cursor) model.Meta {
	return model.Meta{
		RequestID: requestid.FromContext(c), // atau ambil dari c.Locals / header
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Cursor:    cursor,
	}
}

func Success[T any](c fiber.Ctx, status int, data T) error {
	return c.Status(status).JSON(model.Envelope[T]{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c, nil),
	})
}

func NoContent(c fiber.Ctx) error {
	return c.Status(fiber.StatusNoContent).JSON(model.Envelope[interface{}]{
		Success: true,
		Data:    nil,
		Meta:    buildMeta(c, nil),
	})
}

func SuccessList[T any](c fiber.Ctx, status int, data []T, cursor *model.Cursor) error {
	return c.Status(status).JSON(model.Envelope[[]T]{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c, cursor),
	})
}

func Fail(c fiber.Ctx, status int, code, msg string, details any) error {
	return c.Status(status).JSON(model.ErrorEnvelope{
		Success: false,
		Error:   &model.ErrorInfo{Code: code, Message: msg, Details: details},
		Meta:    buildMeta(c, nil),
	})
}

func GetIdParam(c fiber.Ctx) (*uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return nil, errors.New("INVALID_ID_FORMAT")
	}

	return &id, nil
}
