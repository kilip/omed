package httpx

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/api/internal/core"
)

// IDType represents supported identifier types extracted from path parameters (either int64 or uuid.UUID).
type IDType interface {
	int64 | uuid.UUID
}

// GetID extracts and parses the "id" route parameter from the Fiber context into the target IDType.
// It returns core.ErrInvalidID if the parameter is absent, undefined, or fails type parsing.
func GetID[T IDType](c fiber.Ctx) (T, error) {
	var zero T

	raw := c.Params("id")
	if raw == "" || raw == "undefined" {
		return zero, core.ErrInvalidID
	}

	var res any
	switch any(zero).(type) {
	case int64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return zero, core.ErrInvalidID
		}
		res = v
	case uuid.UUID:
		v, err := uuid.Parse(raw)
		if err != nil {
			return zero, core.ErrInvalidID
		}
		res = v
	}

	return res.(T), nil
}
