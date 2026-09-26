package middleware

import (
	"encoding/json"
	"log/slog"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
)

func UserInjector(c fiber.Ctx) error {
	token := jwtware.FromContext(c)
	claims := token.Claims.(jwt.MapClaims)
	b, err := json.Marshal(claims)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	var user model.AuthenticatedUser
	if err := json.Unmarshal(b, &user); err != nil {
		return fiber.ErrInternalServerError
	}
	slog.Info("user authenticated", "user", user)
	c.Locals(http.AUTHENTICATED_USER_KEY, user)
	return c.Next()
}
