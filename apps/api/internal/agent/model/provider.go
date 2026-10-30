package model

import (
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/llmx"
)

// CreateProviderRequest defines payload to create a new provider.
type CreateProviderRequest struct {
	Name         string             `json:"name" validate:"required,min=1,max=100"`
	Kind         string             `json:"kind" validate:"required,oneof=openai_compat anthropic gemini"`
	BaseURL      string             `json:"baseUrl"`
	Model        string             `json:"model" validate:"required,min=1,max=200"`
	APIKey       string             `json:"apiKey" validate:"required"`
	Capabilities *llmx.Capabilities `json:"capabilities,omitempty"`
	IsDefault    bool               `json:"isDefault"`
}

// UpdateProviderRequest defines payload to update an existing provider.
type UpdateProviderRequest struct {
	Name         *string            `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	BaseURL      *string            `json:"baseUrl,omitempty"`
	Model        *string            `json:"model,omitempty" validate:"omitempty,min=1,max=200"`
	APIKey       *string            `json:"apiKey,omitempty" validate:"omitempty"`
	Capabilities *llmx.Capabilities `json:"capabilities,omitempty"`
}

// TestProviderRequest defines payload to test provider configuration.
type TestProviderRequest struct {
	Kind         string             `json:"kind" validate:"required,oneof=openai_compat anthropic gemini"`
	BaseURL      string             `json:"baseUrl"`
	Model        string             `json:"model" validate:"required,min=1,max=200"`
	APIKey       string             `json:"apiKey" validate:"required"`
	Capabilities *llmx.Capabilities `json:"capabilities,omitempty"`
}

// TestResult represents the outcome of a provider connection test.
type TestResult struct {
	OK        bool   `json:"ok"`
	Code      string `json:"code,omitempty"`
	LatencyMs int64  `json:"latencyMs"`
}

// ProviderResponse represents API response view of a provider.
type ProviderResponse struct {
	ID           int64              `json:"id"`
	Name         string             `json:"name"`
	Kind         string             `json:"kind"`
	BaseURL      string             `json:"baseUrl"`
	Model        string             `json:"model"`
	HasKey       bool               `json:"hasKey"`
	KeyHint      string             `json:"keyHint"`
	IsDefault    bool               `json:"isDefault"`
	Capabilities *llmx.Capabilities `json:"capabilities,omitempty"`
	AuditMeta    base.AuditMeta     `json:"auditMeta"`
}
