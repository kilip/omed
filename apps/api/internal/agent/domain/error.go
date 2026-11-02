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
	// ErrInvalidProviderName indicates the provider name is blank or exceeds allowed length.
	ErrInvalidProviderName = errors.New("INVALID_PROVIDER_NAME")
	// ErrInvalidModel indicates the model name is blank or exceeds allowed length.
	ErrInvalidModel = errors.New("INVALID_MODEL")
	// ErrInvalidProviderKind indicates the provider kind is not supported.
	ErrInvalidProviderKind = errors.New("INVALID_PROVIDER_KIND")

	// ErrFinancialAgentDisabled indicates the financial agent is disabled for the workspace.
	ErrFinancialAgentDisabled = errors.New("FINANCIAL_AGENT_DISABLED")
	// ErrFinancialAgentNoProvider indicates no suitable provider is configured for the financial agent.
	ErrFinancialAgentNoProvider = errors.New("FINANCIAL_AGENT_NO_PROVIDER")
	// ErrUnsupportedDocument indicates the document format is not supported by the configured provider.
	ErrUnsupportedDocument = errors.New("UNSUPPORTED_DOCUMENT")
	// ErrAnalysisRateLimited indicates the financial analysis rate limit was exceeded.
	ErrAnalysisRateLimited = errors.New("RATE_LIMITED")
	// ErrAnalysisFailed indicates provider error or malformed model output during analysis.
	ErrAnalysisFailed = errors.New("FINANCIAL_ANALYSIS_FAILED")
	// ErrAnalysisTimeout indicates the financial analysis timed out.
	ErrAnalysisTimeout = errors.New("FINANCIAL_ANALYSIS_TIMEOUT")
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
	base.AddToErrorMap(ErrInvalidProviderName, fiber.StatusUnprocessableEntity, "name must be between 1 and 100 characters")
	base.AddToErrorMap(ErrInvalidModel, fiber.StatusUnprocessableEntity, "model must be between 1 and 200 characters")
	base.AddToErrorMap(ErrInvalidProviderKind, fiber.StatusUnprocessableEntity, "invalid provider kind")

	base.AddToErrorMap(ErrFinancialAgentDisabled, fiber.StatusConflict, "financial agent is disabled for this workspace")
	base.AddToErrorMap(ErrFinancialAgentNoProvider, fiber.StatusUnprocessableEntity, "no suitable provider configured for financial agent")
	base.AddToErrorMap(ErrUnsupportedDocument, fiber.StatusUnprocessableEntity, "unsupported document format for this provider")
	base.AddToErrorMap(ErrAnalysisRateLimited, fiber.StatusTooManyRequests, "financial analysis rate limit exceeded")
	base.AddToErrorMap(ErrAnalysisFailed, fiber.StatusBadGateway, "financial analysis failed")
	base.AddToErrorMap(ErrAnalysisTimeout, fiber.StatusGatewayTimeout, "financial analysis timed out")
}
