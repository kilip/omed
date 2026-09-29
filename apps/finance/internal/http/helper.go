package http

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/kilip/omed/finance/internal/shared/authz"
)

func RequirePermission(obj authz.Resource, act authz.Action) fiber.Handler {
	return func(c fiber.Ctx) error {
		enforcer := authz.GetEnforcer()
		user := shared.UserFromContext(c)

		for _, role := range user.WorkspaceRoles {
			ok, _ := enforcer.Enforce(string(role), user.WorkspaceID.String(), string(obj), string(act))
			if ok {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
}

func GetID(c fiber.Ctx) (*uuid.UUID, error) {
	strID := c.Params("id", "undefined")
	if strID == "undefined" {
		return nil, shared.ErrInvalidID
	}

	id, err := uuid.Parse(strID)
	if err != nil {
		return nil, shared.ErrInvalidID
	}
	return &id, nil
}

func buildMeta(c fiber.Ctx, cursor string) model.Meta {
	return model.Meta{
		RequestID: requestid.FromContext(c),
		Timestamp: time.Now(),
		Cursor:    cursor,
	}
}

func Success[T any](c fiber.Ctx, status int, data T) error {
	return c.Status(status).JSON(model.WebResponse[T]{
		Data: data,
		Meta: buildMeta(c, ""),
	})
}

func WithCursor[T any](c fiber.Ctx, data T, cursor string) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse[T]{
		Data: data,
		Meta: buildMeta(c, cursor),
	})
}

func OK[T any](c fiber.Ctx, data T) error {
	return Success(c, fiber.StatusOK, data)
}

func Created[T any](c fiber.Ctx, data T) error {
	return Success(c, fiber.StatusCreated, data)
}

func BadRequest(c fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
		Errors: err,
		Meta:   buildMeta(c, "nil"),
	})
}
