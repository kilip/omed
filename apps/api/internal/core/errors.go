package core

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

// ErrorItem represents a mapped error containing HTTP status code, machine error code, and friendly message.
type ErrorItem struct {
	err     error
	Status  int
	Code    string
	Message string
}

var errorMaps []ErrorItem

// AddToErrorMap registers an error mapping between an error sentinel, an HTTP status code, and a description.
func AddToErrorMap(err error, status int, msg string) {
	errorMaps = append(errorMaps, ErrorItem{
		err:     err,
		Status:  status,
		Code:    err.Error(),
		Message: msg,
	})
}

var (
	// ErrUnknown indicates an unexpected internal server error.
	ErrUnknown = errors.New("UNKNOWN")
	// ErrInvalidUUID indicates that an invalid UUID format was provided.
	ErrInvalidUUID = errors.New("INVALID_UUID")
	// ErrInvalidID indicates that an invalid identifier was provided.
	ErrInvalidID = errors.New("INVALID_ID")
	// ErrInvalidCursor indicates that a pagination cursor was malformed.
	ErrInvalidCursor = errors.New("INVALID_CURSOR")
	// ErrUnauthorized indicates missing or invalid authentication credentials.
	ErrUnauthorized = errors.New("UNAUTHORIZED")
	// ErrNotFound indicates that the requested resource could not be found.
	ErrNotFound = errors.New("NOT_FOUND")
)

func init() {
	AddToErrorMap(ErrUnknown, fiber.StatusInternalServerError, "unknown error")
	AddToErrorMap(ErrInvalidID, fiber.StatusBadRequest, "invalid id")
	AddToErrorMap(ErrInvalidCursor, fiber.StatusBadRequest, "invalid cursor")
	AddToErrorMap(ErrInvalidUUID, fiber.StatusBadRequest, "invalid uuid")
	AddToErrorMap(ErrUnauthorized, fiber.StatusUnauthorized, "unauthorized access")
	AddToErrorMap(ErrNotFound, fiber.StatusNotFound, "not found")
}

// MapError translates a standard Go error into an ErrorItem by looking up registered error mappings.
// If the error does not match any registered sentinel, ErrUnknown is returned.
func MapError(err error) ErrorItem {
	for _, item := range errorMaps {
		if errors.Is(err, item.err) {
			return item
		}
	}

	return MapError(ErrUnknown)
}

// IsNotFound checks whether an error corresponds to ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
