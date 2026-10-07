package config

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
)

type structValidator struct{ v *validator.Validate }

// Dipanggil otomatis oleh c.Bind().Body/Query/URI
func (s *structValidator) Validate(out any) error { return s.v.Struct(out) }

func GetFiber(config Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		StructValidator: &structValidator{validator.New(validator.WithRequiredStructEnabled())},
		ErrorHandler:    ErrorHandler(logger),
	})

	return app
}

func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := http.MapError(err)
		if status >= 500 {
			log.ErrorContext(c, "unhandled error", "error", err.Error())
		}
		return c.Status(status).JSON(model.ErrorResponse{
			Error: body,
			Meta:  http.NewMeta(c, ""),
		})
	}
}
