// Package jwtx provides JSON Web Token (JWT) verification and user context extraction middlewares using JWKS.
package jwtx

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/kilip/omed/api/internal/base"
)

// Handler initializes and returns a Fiber JWT authentication middleware backed by a remote JWKS endpoint.
// It bypasses OPTIONS requests and paths starting with "/internal".
func Handler(jwksUrl string, logger *slog.Logger) (fiber.Handler, error) {
	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksUrl})
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS from %s: %w", jwksUrl, err)
	}

	config := jwtware.Config{
		Next: func(c fiber.Ctx) bool {
			return c.Method() == fiber.MethodOptions ||
				strings.HasPrefix(c.Path(), "/internal") ||
				strings.HasPrefix(c.Path(), "/auth/") ||
				c.Path() == "/auth"
		},
		KeyFunc:   jwks.KeyfuncCtx(context.Background()),
		Extractor: extractors.FromAuthHeader("Bearer"),
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logger.WarnContext(c, "JWKS Error", "error", err.Error())
			return base.ErrUnauthorized
		},
	}

	return jwtware.New(config), nil
}
