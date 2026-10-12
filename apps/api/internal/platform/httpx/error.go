package httpx

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core"
)

// MapError converts standard domain/validation/framework errors into an HTTP status and ErrorBody.
func MapError(err error) (int, ErrorBody) {
	var ve validator.ValidationErrors

	if errors.As(err, &ve) {
		fields := make([]FieldError, 0, len(ve))
		for _, f := range ve {
			fields = append(fields, FieldError{Field: f.Field(), Rule: f.Tag()})
		}
		return fiber.StatusUnprocessableEntity, ErrorBody{Code: "VALIDATION_FAILED", Message: "validation failed", Fields: fields}
	}

	item := core.MapError(err)
	return item.Status, ErrorBody{Code: item.Code, Message: item.Message}
}
