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
	case errors.Is(err, shared.ErrInvalidDateRange):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_DATE_RANGE", Message: "end_date must be after start_date"}
	case errors.Is(err, shared.ErrPeriodOverlap):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PERIOD_OVERLAP", Message: "period dates overlap with existing period"}
	case errors.Is(err, shared.ErrInvalidStatusTransition):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_STATUS_TRANSITION", Message: "invalid status transition"}
	case errors.Is(err, shared.ErrUnbalancedEntry):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "UNBALANCED_ENTRY", Message: "entry total debit must equal total credit"}
	case errors.Is(err, shared.ErrMinPostingsCount):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "MIN_POSTINGS_COUNT", Message: "entry must have at least 2 posting lines"}
	case errors.Is(err, shared.ErrPeriodClosed):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PERIOD_CLOSED", Message: "entry date must fall within an open ledger period"}
	case errors.Is(err, shared.ErrInactiveAccount):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INACTIVE_ACCOUNT", Message: "cannot post to inactive account"}
	case errors.Is(err, shared.ErrSameCurrencyNotAllowed):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "SAME_CURRENCY_NOT_ALLOWED", Message: "from_currency and to_currency cannot be equal"}
	case errors.Is(err, shared.ErrInvalidExchangeRate):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_EXCHANGE_RATE", Message: "exchange rate must be greater than zero"}
	case errors.Is(err, shared.ErrAccountsNotEmpty):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ACCOUNTS_NOT_EMPTY", Message: "cannot seed accounts when accounts already exist"}
	case errors.Is(err, shared.ErrEntriesNotEmpty):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ENTRIES_NOT_EMPTY", Message: "cannot seed accounts when journal entries exist"}
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
