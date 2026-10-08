package middleware

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/core"
)

type AuthService interface {
	CheckSnapshot(ctx context.Context, user core.AuthenticatedUser) error
}

func UserSnapshotMiddleware(svc AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		user := core.UserFromContext(c)
		if err := svc.CheckSnapshot(c, user); err != nil {
			return err
		}
		return c.Next()
	}
}
