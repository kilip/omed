package jwtx

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/api/internal/base"
)

// UserInjector returns a Fiber middleware that validates issuer and audience claims
// on the verified JWT token, deserializes user information, and injects the resulting
// base.UserContext into the Fiber request context locals.
func UserInjector(issuer string, restAudience string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions || strings.HasPrefix(c.Path(), "/internal") {
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

		if aud, err := claims.GetAudience(); err != nil || !slices.Contains(aud, expectedAudience) {
			return fiber.ErrUnauthorized
		}

		b, err := json.Marshal(claims)
		if err != nil {
			return fiber.ErrUnauthorized
		}

		var user base.UserContext
		if err := json.Unmarshal(b, &user); err != nil {
			return fiber.ErrInternalServerError
		}

		if sub, err := claims.GetSubject(); err == nil && sub != "" {
			id, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				return fiber.ErrUnauthorized
			}
			user.ID = id
		}

		if ev, ok := claims["email_verified"].(bool); ok {
			user.EmailVerified = ev
		} else if ev, ok := claims["emailVerified"].(bool); ok {
			user.EmailVerified = ev
		}

		c.Locals(base.USER_CONTEXT_KEY, user)

		return c.Next()
	}

}
