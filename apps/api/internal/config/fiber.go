package config

import (
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
	slogfiber "github.com/samber/slog-fiber"
)

type structValidator struct{ v *validator.Validate }

func (s *structValidator) Validate(out any) error { return s.v.Struct(out) }

func isSensitiveRoute(c fiber.Ctx) bool {
	path := c.Path()
	method := c.Method()

	if path == "/auth" || strings.HasPrefix(path, "/auth/") {
		return true
	}

	if method == fiber.MethodPost && strings.HasPrefix(path, "/workspaces/") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 3 && parts[2] == "invitations" {
			return true
		}
		if len(parts) == 5 && parts[2] == "invitations" && parts[4] == "send" {
			return true
		}
	}

	return false
}

// GetFiber creates and configures the root Fiber HTTP application instance,
// including timeouts, security headers, rate limiting, struct validation, custom error handling, CORS origins, recovery, and structured logging.
func GetFiber(cfg Config, logger *slog.Logger) *fiber.App {
	maxSizeMB := cfg.AttachmentMaxSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	bodyLimit := int(maxSizeMB*1024*1024) + 1024*1024

	readTimeout := cfg.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 10 * time.Second
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}
	idleTimeout := cfg.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 120 * time.Second
	}

	app := fiber.New(fiber.Config{
		BodyLimit:        bodyLimit,
		ReadTimeout:      readTimeout,
		WriteTimeout:     writeTimeout,
		IdleTimeout:      idleTimeout,
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true, Private: true, LinkLocal: true, Proxies: []string{"0.0.0.0", "127.0.0.1", "::1"}},
		StructValidator:  &structValidator{validator.New(validator.WithRequiredStructEnabled())},
		ErrorHandler:     ErrorHandler(logger),
	})

	// Security headers (helmet)
	helmetCfg := helmet.Config{
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
		XSSProtection:      "0",
		ReferrerPolicy:     "no-referrer",
	}
	if cfg.Env == "production" || strings.HasPrefix(cfg.ApiUrl, "https://") {
		helmetCfg.HSTSMaxAge = 31536000
		helmetCfg.HSTSPreloadEnabled = true
	}
	app.Use(helmet.New(helmetCfg))

	// CORS config
	origins := cfg.TrustedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000"}
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	app.Use(recover.New())

	// Rate Limiting
	if cfg.RateLimitEnabled {
		sensitiveMax := cfg.RateLimitSensitiveMax
		if sensitiveMax <= 0 {
			sensitiveMax = 10
		}
		sensitiveExp := cfg.RateLimitSensitiveExpiration
		if sensitiveExp <= 0 {
			sensitiveExp = 1 * time.Minute
		}

		app.Use(limiter.New(limiter.Config{
			Max:        sensitiveMax,
			Expiration: sensitiveExp,
			KeyGenerator: func(c fiber.Ctx) string {
				return c.IP()
			},
			Next: func(c fiber.Ctx) bool {
				if !cfg.RateLimitEnabled {
					return true
				}
				if c.Get("X-Skip-Rate-Limit") == "true" {
					return true
				}
				if cfg.Env == "test" && c.Get("X-Test-Rate-Limit") != "true" {
					return true
				}
				return !isSensitiveRoute(c)
			},
			LimitReached: func(c fiber.Ctx) error {
				return c.Status(fiber.StatusTooManyRequests).JSON(httpx.ErrorResponse{
					Error: httpx.ErrorBody{
						Code:    "TOO_MANY_REQUESTS",
						Message: "Too many requests, please try again later",
					},
					Meta: httpx.NewMeta(c, ""),
				})
			},
		}))

		generalMax := cfg.RateLimitGeneralMax
		if generalMax <= 0 {
			generalMax = 100
		}
		generalExp := cfg.RateLimitGeneralExpiration
		if generalExp <= 0 {
			generalExp = 1 * time.Minute
		}

		app.Use(limiter.New(limiter.Config{
			Max:        generalMax,
			Expiration: generalExp,
			KeyGenerator: func(c fiber.Ctx) string {
				return c.IP()
			},
			Next: func(c fiber.Ctx) bool {
				if !cfg.RateLimitEnabled {
					return true
				}
				if c.Get("X-Skip-Rate-Limit") == "true" {
					return true
				}
				if cfg.Env == "test" && c.Get("X-Test-Rate-Limit") != "true" {
					return true
				}
				path := c.Path()
				if path == "/healthz" || path == "/readyz" {
					return true
				}
				if c.Method() == fiber.MethodOptions {
					return true
				}
				if isSensitiveRoute(c) {
					return true
				}
				return false
			},
			LimitReached: func(c fiber.Ctx) error {
				return c.Status(fiber.StatusTooManyRequests).JSON(httpx.ErrorResponse{
					Error: httpx.ErrorBody{
						Code:    "TOO_MANY_REQUESTS",
						Message: "Too many requests, please try again later",
					},
					Meta: httpx.NewMeta(c, ""),
				})
			},
		}))
	}

	app.Use(slogfiber.New(logger))

	return app
}

// ErrorHandler returns a Fiber custom error handler that maps application and validation errors
// to consistent JSON error responses, logging unknown errors.
func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := httpx.MapError(err)
		if body.Code == base.ErrUnknown.Error() {
			log.ErrorContext(c, "unknown error", "error", err.Error())
		}
		return c.Status(status).JSON(httpx.ErrorResponse{
			Error: body,
			Meta:  httpx.NewMeta(c, ""),
		})
	}
}
