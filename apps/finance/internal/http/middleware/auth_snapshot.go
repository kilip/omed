package middleware

import (
	"context"
	"encoding/json"
	"slices"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/shared"
)

type AuthService interface {
	CheckSnapshot(ctx context.Context, user shared.AuthenticatedUser) error
}

func UserInjector(issuer string) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := jwtware.FromContext(c)
		if token == nil {
			return fiber.ErrUnauthorized
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.ErrUnauthorized
		}

		if iss, err := claims.GetIssuer(); err != nil || iss != issuer {
			return fiber.ErrUnauthorized
		}
		if aud, err := claims.GetAudience(); err != nil || !slices.Contains(aud, issuer) {
			return fiber.ErrUnauthorized
		}

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

}

func AuthSnapshotMiddleware(svc AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := shared.UserFromContext(c)
		svc.CheckSnapshot(c, user)
		return c.Next()
	}
}
