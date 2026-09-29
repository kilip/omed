package config

import (
	"errors"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type structValidator struct{ v *validator.Validate }

// Dipanggil otomatis oleh c.Bind().Body/Query/URI
func (s *structValidator) Validate(out any) error { return s.v.Struct(out) }

func GetFiber(log *slog.Logger) *fiber.App {
	return fiber.New(fiber.Config{
		StructValidator: &structValidator{validator.New(validator.WithRequiredStructEnabled())},
		ErrorHandler:    ErrorHandler(log),
	})
}

func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, body := mapError(err)
		if status >= 500 {
			log.ErrorContext(c, "unhandled error", "error", err.Error())
		}
		return c.Status(status).JSON(model.ErrorResponse{
			Error: body,
			Meta:  http.NewMeta(c, ""),
		})
	}
}

func mapError(err error) (int, model.ErrorBody) {
	var ve validator.ValidationErrors
	var fe *fiber.Error

	switch {
	case errors.Is(err, shared.ErrItemNotFound):
		return fiber.StatusNotFound, model.ErrorBody{Code: "NOT_FOUND", Message: "resource not found"}
	case errors.Is(err, shared.ErrInvalidID):
		return fiber.StatusBadRequest, model.ErrorBody{Code: "INVALID_ID", Message: "invalid id format"}
	case errors.As(err, &ve):
		fields := make([]model.FieldError, 0, len(ve))
		for _, f := range ve {
			fields = append(fields, model.FieldError{Field: f.Field(), Rule: f.Tag()})
		}
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "VALIDATION_FAILED", Message: "validation failed", Fields: fields}
	case errors.As(err, &fe):
		return fe.Code, model.ErrorBody{Code: "HTTP_ERROR", Message: fe.Message}
	default:
		// jangan bocorin detail internal ke client
		return fiber.StatusInternalServerError, model.ErrorBody{Code: "INTERNAL", Message: "internal server error"}
	}
}
