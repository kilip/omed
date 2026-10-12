package app

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/httpx"
	slogfiber "github.com/samber/slog-fiber"
)

type structValidator struct{ v *validator.Validate }

func (s *structValidator) Validate(out any) error { return s.v.Struct(out) }

func GetFiber(cfg core.Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		StructValidator: &structValidator{validator.New(validator.WithRequiredStructEnabled())},
		ErrorHandler:    ErrorHandler(logger),
	})

	// cors config
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.TrustedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))

	app.Use(recover.New())
	app.Use(slogfiber.New(logger))

	return app
}

func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := httpx.MapError(err)
		if body.Code == core.ErrUnknown.Error() {
			log.ErrorContext(c, "unknown error", "error", err.Error())
		}
		return c.Status(status).JSON(httpx.ErrorResponse{
			Error: body,
			Meta:  httpx.NewMeta(c, ""),
		})
	}
}
