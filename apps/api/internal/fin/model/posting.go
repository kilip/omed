// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/kilip/omed/api/internal/base"
)

// CreatePostingRequest defines the payload required to create a posting line.
type CreatePostingRequest struct {
	AccountID   int64   `json:"accountId" validate:"required"`
	Direction   string  `json:"direction" validate:"required,oneof=DEBIT CREDIT" example:"DEBIT"`
	Amount      float64 `json:"amount" validate:"required,gt=0" example:"100000"`
	Description string  `json:"description,omitempty"`
}

// PostingResponse represents posting line metadata returned by API endpoints.
type PostingResponse struct {
	ID          int64          `json:"id"`
	WorkspaceID int64          `json:"workspaceId"`
	EntryID     int64          `json:"entryId"`
	AccountID   int64          `json:"accountId"`
	Direction   string         `json:"direction" example:"DEBIT"`
	Amount      float64        `json:"amount" example:"100000"`
	BaseAmount  float64        `json:"baseAmount" example:"100000"`
	Position    int            `json:"position" example:"0"`
	Description string         `json:"description,omitempty"`
	AuditMeta   base.AuditMeta `json:"auditMeta"`
}
