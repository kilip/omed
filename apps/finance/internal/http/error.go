package http

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
)

// MapError converts standard domain/validation/framework errors into an HTTP status and ErrorBody.
func MapError(err error) (int, model.ErrorBody) {
	var ve validator.ValidationErrors
	var fe *fiber.Error

	switch {
	case errors.Is(err, core.ErrItemNotFound):
		return fiber.StatusNotFound, model.ErrorBody{Code: "NOT_FOUND", Message: "resource not found"}
	case errors.Is(err, core.ErrForbidden):
		return fiber.StatusForbidden, model.ErrorBody{Code: "FORBIDDEN", Message: "access forbidden"}
	case errors.Is(err, core.ErrInvalidID) || (err != nil && strings.Contains(err.Error(), "invalid UUID")):
		return fiber.StatusBadRequest, model.ErrorBody{Code: "INVALID_ID", Message: "invalid id format"}
	case errors.Is(err, core.ErrInvalidDateRange):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_DATE_RANGE", Message: "end_date must be after start_date"}
	case errors.Is(err, core.ErrPeriodOverlap):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PERIOD_OVERLAP", Message: "period dates overlap with existing period"}
	case errors.Is(err, core.ErrInvalidStatusTransition):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_STATUS_TRANSITION", Message: "invalid status transition"}
	case errors.Is(err, core.ErrUnbalancedEntry):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "UNBALANCED_ENTRY", Message: "entry total debit must equal total credit"}
	case errors.Is(err, core.ErrMinPostingsCount):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "MIN_POSTINGS_COUNT", Message: "entry must have at least 2 posting lines"}
	case errors.Is(err, core.ErrPeriodClosed):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PERIOD_CLOSED", Message: "entry date must fall within an open ledger period"}
	case errors.Is(err, core.ErrInactiveAccount):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INACTIVE_ACCOUNT", Message: "cannot post to inactive account"}
	case errors.Is(err, core.ErrSameCurrencyNotAllowed):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "SAME_CURRENCY_NOT_ALLOWED", Message: "from_currency and to_currency cannot be equal"}
	case errors.Is(err, core.ErrInvalidExchangeRate):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_EXCHANGE_RATE", Message: "exchange rate must be greater than zero"}
	case errors.Is(err, core.ErrAccountsNotEmpty):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ACCOUNTS_NOT_EMPTY", Message: "cannot seed accounts when accounts already exist"}
	case errors.Is(err, core.ErrEntriesNotEmpty):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ENTRIES_NOT_EMPTY", Message: "cannot seed accounts when journal entries exist"}
	case errors.Is(err, core.ErrInvalidFileType):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_FILE_TYPE", Message: "unsupported attachment file type"}
	case errors.Is(err, core.ErrFileTooLarge):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "FILE_TOO_LARGE", Message: "attachment file size exceeds limit"}
	case errors.Is(err, core.ErrAttachmentNotPending):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ATTACHMENT_NOT_PENDING", Message: "attachment is not in pending status"}
	case errors.Is(err, core.ErrAttachmentObjectNotFound):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ATTACHMENT_OBJECT_NOT_FOUND", Message: "uploaded object not found in storage"}
	case errors.Is(err, core.ErrPeriodLocked):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PERIOD_LOCKED", Message: "cannot modify attachments in locked ledger period"}
	case errors.Is(err, core.ErrInvalidAccountType):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_ACCOUNT_TYPE", Message: "account type is invalid for this purpose"}
	case errors.Is(err, core.ErrMissingAccountMapping):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "MISSING_ACCOUNT_MAPPING", Message: "required account mapping is not configured"}
	case errors.Is(err, core.ErrContactInUse):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "CONTACT_IN_USE", Message: "contact cannot be deleted because it is in use"}
	case errors.Is(err, core.ErrTaxRateInUse):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "TAX_RATE_IN_USE", Message: "tax rate cannot be deleted because it is in use"}
	case errors.Is(err, core.ErrInvalidTaxRate):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVALID_TAX_RATE", Message: "tax rate must be between 0 and 100"}
	case errors.Is(err, core.ErrInvoiceNotDraft):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVOICE_NOT_DRAFT", Message: "invoice is not in draft status"}
	case errors.Is(err, core.ErrInvoiceNoLines):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVOICE_NO_LINES", Message: "invoice must have at least one line"}
	case errors.Is(err, core.ErrContactKindMismatch):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "CONTACT_KIND_MISMATCH", Message: "contact kind does not match invoice type"}
	case errors.Is(err, core.ErrTaxKindMismatch):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "TAX_KIND_MISMATCH", Message: "tax rate kind does not match invoice type"}
	case errors.Is(err, core.ErrInvoiceHasPayments):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVOICE_HAS_PAYMENTS", Message: "invoice cannot be voided because it has payments"}
	case errors.Is(err, core.ErrInvoiceAlreadyVoid):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVOICE_ALREADY_VOID", Message: "invoice is already void"}
	case errors.Is(err, core.ErrInvoiceNotIssued):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVOICE_NOT_ISSUED", Message: "invoice is not in issued status"}
	case errors.Is(err, core.ErrEntryLinked):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "ENTRY_LINKED", Message: "entry is linked to an invoice or payment and cannot be modified"}
	case errors.Is(err, core.ErrOverpayment):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "OVERPAYMENT", Message: "payment allocation exceeds invoice outstanding amount"}
	case errors.Is(err, core.ErrPaymentMismatch):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PAYMENT_MISMATCH", Message: "payment amount does not match sum of allocations"}
	case errors.Is(err, core.ErrCurrencyMismatch):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "CURRENCY_MISMATCH", Message: "payment currency does not match invoice currency"}
	case errors.Is(err, core.ErrPaymentAlreadyVoid):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PAYMENT_ALREADY_VOID", Message: "payment is already void"}
	case errors.Is(err, core.ErrPaymentNotPosted):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "PAYMENT_NOT_POSTED", Message: "payment is not in posted status"}
	case errors.Is(err, core.ErrDuplicateReference):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "DUPLICATE_REFERENCE", Message: "reference is already in use by another bill for this vendor"}
	case errors.Is(err, core.ErrInvoiceNotSendable):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "INVOICE_NOT_SENDABLE", Message: "invoice cannot be sent"}
	case errors.Is(err, core.ErrExchangeRateMissing):
		return fiber.StatusUnprocessableEntity, model.ErrorBody{Code: "EXCHANGE_RATE_MISSING", Message: "exchange rate not found for currency conversion"}
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
