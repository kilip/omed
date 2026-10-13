package core

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

type ErrorItem struct {
	err     error
	Status  int
	Code    string
	Message string
}

var errorMaps []ErrorItem

func AddToErrorMap(err error, status int, msg string) {
	errorMaps = append(errorMaps, ErrorItem{
		err:     err,
		Status:  status,
		Code:    err.Error(),
		Message: msg,
	})
}

var (
	ErrUnknown       = errors.New("UNKNOWN")
	ErrInvalidUUID   = errors.New("INVALID_UUID")
	ErrInvalidID     = errors.New("INVALID_ID")
	ErrInvalidCursor = errors.New("INVALID_CURSOR")
	ErrUnauthorized  = errors.New("UNAUTHORIZED")
	ErrNotFound      = errors.New("NOT_FOUND")
)

func init() {
	AddToErrorMap(ErrUnknown, fiber.StatusInternalServerError, "unknown error")
	AddToErrorMap(ErrInvalidID, fiber.StatusBadRequest, "invalid id")
	AddToErrorMap(ErrInvalidCursor, fiber.StatusBadRequest, "invalid cursor")
	AddToErrorMap(ErrInvalidUUID, fiber.StatusBadRequest, "invalid uuid")
	AddToErrorMap(ErrUnauthorized, fiber.StatusUnauthorized, "unauthorized access")
	AddToErrorMap(ErrNotFound, fiber.StatusNotFound, "not found")
}

func MapError(err error) ErrorItem {
	for _, item := range errorMaps {
		if errors.Is(err, item.err) {
			return item
		}
	}

	return MapError(ErrUnknown)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
