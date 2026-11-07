package config

import (
	"encoding/json"
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

const (
	maxLoggedBodyBytes    = 64 * 1024
	maxMalformedRawBytes  = 1024
	bodyLoggedLocalsKey   = "omed:request_body_logged"
	errorSummaryLocalsKey = "omed:error_summary"
)

var sensitiveFields = map[string]struct{}{
	"password":        {},
	"pass":            {},
	"currentpassword": {},
	"newpassword":     {},
	"secret":          {},
	"secretkey":       {},
	"clientsecret":    {},
	"token":           {},
	"accesstoken":     {},
	"refreshtoken":    {},
	"idtoken":         {},
	"authorization":   {},
	"apikey":          {},
	"code":            {},
	"otp":             {},
	"totp":            {},
	"pin":             {},
	"passphrase":      {},
	"accesskey":       {},
	"credential":      {},
	"credentials":     {},
	"privatekey":      {},
}

func isSensitiveField(key string) bool {
	normalized := strings.ToLower(key)
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	if _, ok := sensitiveFields[normalized]; ok {
		return true
	}
	if strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "privatekey") ||
		strings.Contains(normalized, "accesskey") ||
		strings.Contains(normalized, "passphrase") {
		return true
	}
	return false
}

func sanitizeValue(val any) any {
	switch v := val.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(v))
		for k, item := range v {
			if isSensitiveField(k) {
				sanitized[k] = "[REDACTED]"
			} else {
				sanitized[k] = sanitizeValue(item)
			}
		}
		return sanitized
	case []any:
		sanitized := make([]any, len(v))
		for i, item := range v {
			sanitized[i] = sanitizeValue(item)
		}
		return sanitized
	default:
		return v
	}
}

var sensitiveKeywords = []string{
	"password",
	"secret",
	"token",
	"apikey",
	"api_key",
	"api-key",
	"privatekey",
	"private_key",
	"credential",
	"otp",
	"totp",
	"authorization",
}

func containsSensitiveKeyword(raw string) bool {
	lower := strings.ToLower(raw)
	for _, kw := range sensitiveKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func isSensitivePayloadRoute(c fiber.Ctx) bool {
	if isSensitiveRoute(c) {
		return true
	}
	path := c.Path()
	return strings.Contains(path, "/agent/providers") || strings.Contains(path, "/tokens")
}

func attachRequestBody(c fiber.Ctx) {
	if c.Locals(bodyLoggedLocalsKey) != nil {
		return
	}
	c.Locals(bodyLoggedLocalsKey, true)

	contentType := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	if !strings.HasPrefix(contentType, fiber.MIMEApplicationJSON) {
		return
	}
	body := c.Body()
	if len(body) == 0 || len(body) > maxLoggedBodyBytes {
		return
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err == nil {
		sanitized := sanitizeValue(parsed)
		slogfiber.AddCustomAttributes(c, slog.Any("request_body", sanitized))
	} else {
		raw := string(body)
		if !isSensitivePayloadRoute(c) && !containsSensitiveKeyword(raw) {
			if len(raw) > maxMalformedRawBytes {
				raw = raw[:maxMalformedRawBytes]
			}
			raw = strings.ToValidUTF8(raw, "")
			slogfiber.AddCustomAttributes(c, slog.Any("request_body", map[string]any{
				"_malformed": true,
				"raw":        raw,
			}))
		}
	}
}

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
	maxLimitMB := cfg.AttachmentMaxSizeMB
	if cfg.CMSMediaMaxSizeMB > maxLimitMB {
		maxLimitMB = cfg.CMSMediaMaxSizeMB
	}
	if maxLimitMB <= 0 {
		maxLimitMB = 10
	}
	bodyLimit := int(maxLimitMB*1024*1024) + 1024*1024 // +1MB overhead for multipart headers

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
		ErrorHandler:     ErrorHandler(logger, cfg.LogHTTPBody),
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
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "sentry-trace", "baggage"},
		AllowCredentials: true,
	}))

	slogCfg := slogfiber.DefaultConfig()
	slogCfg.WithCustomMessage = func(c fiber.Ctx, err error) string {
		if summary, ok := c.Locals(errorSummaryLocalsKey).(string); ok && summary != "" {
			return summary
		}
		if c.Response().StatusCode() < 400 {
			return "Incoming request"
		}
		if err != nil {
			return err.Error()
		}
		return "Incoming request"
	}
	app.Use(slogfiber.NewWithConfig(logger, slogCfg))

	if cfg.LogHTTPBody {
		app.Use(func(c fiber.Ctx) error {
			err := c.Next()
			status := c.Response().StatusCode()
			if err != nil {
				status, _ = httpx.MapError(err)
			}
			if status >= 400 {
				attachRequestBody(c)
			}
			return err
		})
	}

	app.Use(recover.New())
	app.Use(SentryMiddleware(cfg))

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
				c.Locals(errorSummaryLocalsKey, "TOO_MANY_REQUESTS: Too many requests, please try again later")
				slogfiber.AddCustomAttributes(c, slog.Group("error",
					slog.String("code", "TOO_MANY_REQUESTS"),
					slog.String("message", "Too many requests, please try again later"),
				))
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
				c.Locals(errorSummaryLocalsKey, "TOO_MANY_REQUESTS: Too many requests, please try again later")
				slogfiber.AddCustomAttributes(c, slog.Group("error",
					slog.String("code", "TOO_MANY_REQUESTS"),
					slog.String("message", "Too many requests, please try again later"),
				))
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

	return app
}

// ErrorHandler returns a Fiber custom error handler that maps application and validation errors
// to consistent JSON error responses, logging unknown errors and attaching structured error details.
func ErrorHandler(log *slog.Logger, logHTTPBody ...bool) fiber.ErrorHandler {
	shouldLogBody := false
	if len(logHTTPBody) > 0 {
		shouldLogBody = logHTTPBody[0]
	}

	return func(c fiber.Ctx, err error) error {
		status, body := httpx.MapError(err)

		c.Locals(errorSummaryLocalsKey, body.Code+": "+body.Message)

		var errorAttrs []any
		errorAttrs = append(errorAttrs,
			slog.String("code", body.Code),
			slog.String("message", body.Message),
		)
		if len(body.Fields) > 0 {
			errorAttrs = append(errorAttrs, slog.Any("fields", body.Fields))
		}
		if (status >= 500 || body.Code == base.ErrUnknown.Error()) && err != nil {
			errorAttrs = append(errorAttrs, slog.String("cause", err.Error()))
			if body.Code == base.ErrUnknown.Error() && log != nil {
				log.ErrorContext(c, "unknown error", "error", err)
			}
		}

		slogfiber.AddCustomAttributes(c, slog.Group("error", errorAttrs...))

		if shouldLogBody && status >= 400 {
			attachRequestBody(c)
		}

		if status >= 500 && err != nil {
			CaptureFiberError(c, err, body.Code)
		}
		return c.Status(status).JSON(httpx.ErrorResponse{
			Error: body,
			Meta:  httpx.NewMeta(c, ""),
		})
	}
}
