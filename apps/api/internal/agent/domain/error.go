package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

var (
	// ErrProviderNotFound indicates the requested provider was not found.
	ErrProviderNotFound = errors.New("PROVIDER_NOT_FOUND")
	// ErrProviderNameTaken indicates the provider name is already in use in the workspace.
	ErrProviderNameTaken = errors.New("PROVIDER_NAME_TAKEN")
	// ErrNoDefaultProvider indicates no default provider is configured for the workspace.
	ErrNoDefaultProvider = errors.New("NO_DEFAULT_PROVIDER")
	// ErrInvalidBaseURL indicates the provider base URL is malformed or invalid.
	ErrInvalidBaseURL = errors.New("INVALID_BASE_URL")
	// ErrSecretKeyMissing indicates the encryption secret key is not configured.
	ErrSecretKeyMissing = errors.New("SECRET_KEY_MISSING")
	// ErrProviderKeyRequired indicates an API key is required for the provider kind.
	ErrProviderKeyRequired = errors.New("PROVIDER_KEY_REQUIRED")
	// ErrProviderKeyUnreadable indicates the stored API key ciphertext could not be decrypted.
	ErrProviderKeyUnreadable = errors.New("PROVIDER_KEY_UNREADABLE")
	// ErrTestRateLimited indicates provider connection test rate limit was exceeded.
	ErrTestRateLimited = errors.New("TEST_RATE_LIMITED")
)

func init() {
	base.AddToErrorMap(ErrProviderNotFound, fiber.StatusNotFound, "provider not found")
	base.AddToErrorMap(ErrProviderNameTaken, fiber.StatusConflict, "provider name already exists in this workspace")
	base.AddToErrorMap(ErrNoDefaultProvider, fiber.StatusUnprocessableEntity, "no default provider configured")
	base.AddToErrorMap(ErrInvalidBaseURL, fiber.StatusUnprocessableEntity, "invalid provider base url")
	base.AddToErrorMap(ErrSecretKeyMissing, fiber.StatusServiceUnavailable, "secret key is not configured")
	base.AddToErrorMap(ErrProviderKeyRequired, fiber.StatusUnprocessableEntity, "api key is required for this provider")
	base.AddToErrorMap(ErrProviderKeyUnreadable, fiber.StatusConflict, "failed to decrypt provider api key")
	base.AddToErrorMap(ErrTestRateLimited, fiber.StatusTooManyRequests, "provider test rate limit exceeded")
}
