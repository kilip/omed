// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/kilip/omed/api/internal/base"
)

// CreateTransactionRequest defines the payload required to create a new financial transaction.
type CreateTransactionRequest struct {
	TransactionDate Date                      `json:"transactionDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Description     string                    `json:"description,omitempty"`
	Reference       string                    `json:"reference,omitempty"`
	Currency        string                    `json:"currency,omitempty" validate:"omitempty,len=3" example:"IDR"`
	ExchangeRate    *float64                  `json:"exchangeRate,omitempty" validate:"omitempty,gt=0" example:"1"`
	Entry           *CreateEntryInlineRequest `json:"entry,omitempty"`
}

// UpdateTransactionRequest defines the payload required to update a transaction's header fields.
type UpdateTransactionRequest struct {
	Description *string `json:"description,omitempty"`
	Reference   *string `json:"reference,omitempty"`
}

// SearchTransactionRequest defines request payload for searching transactions in a workspace.
type SearchTransactionRequest struct {
	DateFrom  *Date   `json:"dateFrom,omitempty" swaggertype:"string" example:"2026-01-01"`
	DateTo    *Date   `json:"dateTo,omitempty" swaggertype:"string" example:"2026-01-31"`
	Reference *string `json:"reference,omitempty"`
	Source    *string `json:"source,omitempty" validate:"omitempty,oneof=MANUAL IMPORT SYSTEM"`
}

// TransactionResponse represents financial transaction metadata returned by API endpoints.
type TransactionResponse struct {
	ID              int64           `json:"id"`
	WorkspaceID     int64           `json:"workspaceId"`
	Number          string          `json:"number"`
	TransactionDate Date            `json:"transactionDate" swaggertype:"string" example:"2026-01-01"`
	Description     string          `json:"description,omitempty"`
	Reference       string          `json:"reference,omitempty"`
	Currency        string          `json:"currency" example:"IDR"`
	ExchangeRate    float64         `json:"exchangeRate" example:"1"`
	Source          string          `json:"source" example:"MANUAL"`
	Entries         []EntryResponse `json:"entries,omitempty"`
	AuditMeta       base.AuditMeta  `json:"auditMeta"`
}
