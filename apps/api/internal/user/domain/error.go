// Package domain defines core domain entities and business errors for the user module.
package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

// ErrEmailTaken indicates that the specified email address is already in use by another account.
var ErrEmailTaken = errors.New("USER_EMAIL_TAKEN")

func init() {
	base.AddToErrorMap(ErrEmailTaken, fiber.StatusConflict, "email already used")
}
