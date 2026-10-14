package config

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
	slogfiber "github.com/samber/slog-fiber"
)

type structValidator struct{ v *validator.Validate }

func (s *structValidator) Validate(out any) error { return s.v.Struct(out) }

// GetFiber creates and configures the root Fiber HTTP application instance,
// including struct validation, custom error handling, CORS origins, recovery, and structured logging.
func GetFiber(cfg Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		StructValidator: &structValidator{validator.New(validator.WithRequiredStructEnabled())},
		ErrorHandler:    ErrorHandler(logger),
	})

	// cors config
	origins := cfg.TrustedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:3001"}
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	app.Use(recover.New())
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
