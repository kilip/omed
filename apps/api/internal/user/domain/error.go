package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core"
)

var ErrEmailTaken = errors.New("USER_EMAIL_TAKEN")

func init() {
	core.AddToErrorMap(ErrEmailTaken, fiber.StatusConflict, "email already used")
}
