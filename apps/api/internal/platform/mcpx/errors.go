package mcpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// ErrorDetailer provides additional human-readable context appended to the mapped error message.
type ErrorDetailer interface {
	ErrorDetail() string
}

// ErrResultTooLarge indicates that a tool's serialized output exceeded the size cap.
var ErrResultTooLarge = errors.New("RESULT_TOO_LARGE")

func init() {
	base.AddToErrorMap(ErrResultTooLarge, fiber.StatusRequestEntityTooLarge, "result too large")
}

// ErrorPayload represents the JSON text content embedded inside a tool error result.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// formatToolError formats an error into a JSON string suitable for the client.
func formatToolError(ctx context.Context, err error, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	_, body := httpx.MapError(err)

	if body.Code == base.ErrUnknown.Error() {
		ref := uuid.New().String()
		logger.ErrorContext(ctx, "mcp tool handler failed", "ref", ref, "error", err)
		body.Message = fmt.Sprintf("internal error (ref: %s)", ref)
	} else {
		var detailer ErrorDetailer
		if errors.As(err, &detailer) {
			if d := detailer.ErrorDetail(); d != "" {
				if strings.HasPrefix(d, ":") || strings.HasPrefix(d, " ") {
					body.Message += d
				} else {
					body.Message += ": " + d
				}
			}
		}
	}

	payload := ErrorPayload{
		Code:    body.Code,
		Message: body.Message,
	}

	data, _ := json.Marshal(payload)
	return errors.New(string(data))
}

// newValidationError returns a tool validation error.
func newValidationError(message string) error {
	payload := ErrorPayload{
		Code:    "VALIDATION_FAILED",
		Message: message,
	}
	data, _ := json.Marshal(payload)
	return errors.New(string(data))
}
