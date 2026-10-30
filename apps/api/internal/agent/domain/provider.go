package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/llmx"
)

// Provider represents an AI model provider configuration in the agent domain.
type Provider struct {
	ID           int64              `json:"id"`
	WorkspaceID  int64              `json:"workspaceId"`
	Name         string             `json:"name"`
	Kind         llmx.Kind          `json:"kind"`
	BaseURL      string             `json:"baseUrl"`
	Model        string             `json:"model"`
	KeyHint      string             `json:"keyHint"`
	HasKey       bool               `json:"hasKey"`
	IsDefault    bool               `json:"isDefault"`
	Capabilities *llmx.Capabilities `json:"capabilities"`
	AuditMeta    base.AuditMeta     `json:"auditMeta"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
}
