package middleware

import (
	"encoding/json"
	"slices"
	"strings"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/core"
)

func UserInjector(issuer string, restAudience string, mcpResource string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}
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

		expectedAudience := restAudience
		if strings.HasPrefix(c.Path(), "/mcp") {
			expectedAudience = mcpResource
		}

		if aud, err := claims.GetAudience(); err != nil || !slices.Contains(aud, expectedAudience) {
			return fiber.ErrUnauthorized
		}

		b, err := json.Marshal(claims)
		if err != nil {
			return fiber.ErrUnauthorized
		}

		var user core.AuthenticatedUser
		if err := json.Unmarshal(b, &user); err != nil {
			return fiber.ErrInternalServerError
		}
		c.Locals(core.AUTH_USER_CONTEXT_KEY, user)

		return c.Next()
	}

}
