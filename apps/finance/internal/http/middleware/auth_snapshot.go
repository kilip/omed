package middleware

import (
	"context"
	"encoding/json"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/shared"
)

type AuthService interface {
	CheckSnapshot(ctx context.Context, user shared.AuthenticatedUser) error
}

func UserInjector(c fiber.Ctx) error {
	token := jwtware.FromContext(c)
	claims := token.Claims.(jwt.MapClaims)
	b, err := json.Marshal(claims)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	var user shared.AuthenticatedUser
	if err := json.Unmarshal(b, &user); err != nil {
		return fiber.ErrInternalServerError
	}
	c.Locals(shared.AUTH_USER_CONTEXT_KEY, user)

	return c.Next()
}

func AuthSnapshotMiddleware(svc AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := shared.UserFromContext(c)
		svc.CheckSnapshot(c, user)
		return c.Next()
	}
}
