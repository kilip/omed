// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// CreateEntryRequest defines the payload required to add a journal entry to a transaction.
type CreateEntryRequest struct {
	TransactionID int64                  `json:"transactionId,omitempty"`
	PeriodID      int64                  `json:"periodId" validate:"required"`
	EntryDate     Date                   `json:"entryDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Description   string                 `json:"description,omitempty"`
	Postings      []CreatePostingRequest `json:"postings" validate:"required,min=2,dive"`
}

// CreateEntryInlineRequest defines the payload required to create an entry inline with a transaction.
type CreateEntryInlineRequest struct {
	PeriodID    int64                  `json:"periodId" validate:"required"`
	EntryDate   Date                   `json:"entryDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Description string                 `json:"description,omitempty"`
	Postings    []CreatePostingRequest `json:"postings" validate:"required,min=2,dive"`
}

// UpdateEntryRequest defines the payload required to update a draft entry's description and postings.
type UpdateEntryRequest struct {
	Description *string                `json:"description,omitempty"`
	Postings    []CreatePostingRequest `json:"postings" validate:"required,min=2,dive"`
}

// ReverseEntryRequest defines payload for reversing a posted journal entry.
type ReverseEntryRequest struct {
	EntryDate   *Date  `json:"entryDate,omitempty" swaggertype:"string" example:"2026-01-01"`
	Description string `json:"description,omitempty"`
}

// SearchEntryRequest defines request payload for searching journal entries in a workspace.
type SearchEntryRequest struct {
	PeriodID  *int64  `json:"periodId,omitempty"`
	Status    *string `json:"status,omitempty" validate:"omitempty,oneof=DRAFT POSTED VOID"`
	AccountID *int64  `json:"accountId,omitempty"`
	DateFrom  *Date   `json:"dateFrom,omitempty" swaggertype:"string" example:"2026-01-01"`
	DateTo    *Date   `json:"dateTo,omitempty" swaggertype:"string" example:"2026-01-31"`
}

// EntryResponse represents journal entry metadata returned by API endpoints.
type EntryResponse struct {
	ID           int64             `json:"id"`
	WorkspaceID  int64             `json:"workspaceId"`
	TransactionID int64            `json:"transactionId"`
	PeriodID     int64             `json:"periodId"`
	Number       string            `json:"number"`
	EntryDate    Date              `json:"entryDate" swaggertype:"string" example:"2026-01-01"`
	Description  string            `json:"description,omitempty"`
	Status       string            `json:"status" example:"DRAFT"`
	PostedAt     *time.Time        `json:"postedAt,omitempty"`
	PostedBy     *int64            `json:"postedBy,omitempty"`
	ReversalOfID *int64            `json:"reversalOfId,omitempty"`
	Postings     []PostingResponse `json:"postings,omitempty"`
	AuditMeta    base.AuditMeta    `json:"auditMeta"`
}
