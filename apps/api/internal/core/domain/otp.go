// Package domain defines core domain entities and business errors for the core module.
package domain

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

var (
	// ErrOTPInvalid indicates that the OTP code is invalid or expired.
	ErrOTPInvalid = errors.New("AUTH_OTP_INVALID")
	// ErrOTPUnavailable indicates that email sign-in is not available.
	ErrOTPUnavailable = errors.New("AUTH_OTP_UNAVAILABLE")
)

func init() {
	base.AddToErrorMap(ErrOTPInvalid, fiber.StatusBadRequest, "invalid or expired code")
	base.AddToErrorMap(ErrOTPUnavailable, fiber.StatusServiceUnavailable, "email sign-in is not available")
}

// OTP represents a login one-time password record.
type OTP struct {
	ID         int64
	Email      string
	CodeHash   string
	ExpiresAt  time.Time
	Attempts   int
	ConsumedAt *time.Time
	IPAddress  string
	CreatedAt  time.Time
}
