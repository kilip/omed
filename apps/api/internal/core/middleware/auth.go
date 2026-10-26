// Package middleware provides HTTP middlewares specific to the core module.
package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/service"
)

// AuthConfig defines configuration options for the session authentication middleware.
type AuthConfig struct {
	SessionService service.SessionService
	Logger         *slog.Logger
	CookieName     string
}

// NewAuthMiddleware returns a Fiber handler that authenticates requests using SessionService.
// It bypasses OPTIONS, /healthz, /readyz, /internal/*, /auth, /auth/*, GET /invitations/:token, and unauthenticated GET/HEAD requests to /public/*.
func NewAuthMiddleware(cfg AuthConfig) fiber.Handler {
	cookieName := cfg.CookieName
	if cookieName == "" {
		cookieName = "omed_session"
	}

	return func(c fiber.Ctx) error {
		// Bypass CORS preflight and public routes
		if c.Method() == fiber.MethodOptions ||
			c.Path() == "/healthz" ||
			c.Path() == "/readyz" ||
			strings.HasPrefix(c.Path(), "/internal") ||
			strings.HasPrefix(c.Path(), "/auth/") ||
			c.Path() == "/auth" ||
			strings.HasPrefix(c.Path(), "/test/") ||
			c.Path() == "/test" ||
			((c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead) && strings.HasPrefix(c.Path(), "/public/")) ||
			(c.Method() == fiber.MethodGet && strings.HasPrefix(c.Path(), "/invitations/") && !strings.HasSuffix(c.Path(), "/accept")) {
			return c.Next()
		}

		token := extractToken(c, cookieName)
		if token == "" {
			return base.ErrUnauthorized
		}

		_, usr, err := cfg.SessionService.ValidateSession(c.Context(), token)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.WarnContext(c, "Session validation failed", "error", err.Error())
			}
			return base.ErrUnauthorized
		}

		userCtx := base.UserContext{
			ID:            usr.ID,
			Name:          usr.Name,
			EmailVerified: true,
		}

		c.Locals(base.USER_CONTEXT_KEY, userCtx)
		return c.Next()
	}
}

func extractToken(c fiber.Ctx, cookieName string) string {
	authHeader := c.Get(fiber.HeaderAuthorization)
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if cookie := c.Cookies(cookieName); cookie != "" {
		return cookie
	}
	if cookieName != "omed_session" {
		if cookie := c.Cookies("omed_session"); cookie != "" {
			return cookie
		}
	}
	if cookieName != "token" {
		if cookie := c.Cookies("token"); cookie != "" {
			return cookie
		}
	}
	return ""
}
