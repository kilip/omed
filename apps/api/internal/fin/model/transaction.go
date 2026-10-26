// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"io"

	"github.com/kilip/omed/api/internal/base"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/shopspring/decimal"
)

// CreateTransactionRequest defines the payload required to create a new financial transaction.
type CreateTransactionRequest struct {
	TransactionDate Date                      `json:"transactionDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Description     string                    `json:"description,omitempty"`
	Reference       string                    `json:"reference,omitempty"`
	Currency        string                    `json:"currency,omitempty" validate:"omitempty,len=3" example:"IDR"`
	ExchangeRate    *decimal.Decimal          `json:"exchangeRate,omitempty" swaggertype:"string" example:"1.00000000"`
	Entry           *CreateEntryInlineRequest `json:"entry,omitempty"`
}

// UpdateTransactionRequest defines the payload required to update a transaction's header fields.
type UpdateTransactionRequest struct {
	Description *string `json:"description,omitempty"`
	Reference   *string `json:"reference,omitempty"`
}

// UploadTransactionAttachmentRequest defines the input required to attach a file to a transaction.
type UploadTransactionAttachmentRequest struct {
	WorkspaceID int64
	Name        string
	MimeType    string
	Size        int64
	Reader      io.Reader
}

// SearchTransactionRequest defines request payload for searching transactions in a workspace.
type SearchTransactionRequest struct {
	DateFrom  *Date   `json:"dateFrom,omitempty" swaggertype:"string" example:"2026-01-01"`
	DateTo    *Date   `json:"dateTo,omitempty" swaggertype:"string" example:"2026-01-31"`
	Reference *string `json:"reference,omitempty"`
	Source    *string `json:"source,omitempty" validate:"omitempty,oneof=MANUAL IMPORT SYSTEM"`
	Query     *string `json:"query,omitempty"`
	Cursor    *string `json:"cursor,omitempty"`
	Limit     *int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// TransactionResponse represents financial transaction metadata returned by API endpoints.
type TransactionResponse struct {
	ID              int64                    `json:"id"`
	WorkspaceID     int64                    `json:"workspaceId"`
	Number          string                   `json:"number"`
	TransactionDate Date                     `json:"transactionDate" swaggertype:"string" example:"2026-01-01"`
	Description     string                   `json:"description,omitempty"`
	Reference       string                   `json:"reference,omitempty"`
	Currency        string                   `json:"currency" example:"IDR"`
	ExchangeRate    decimal.Decimal          `json:"exchangeRate" swaggertype:"string" example:"1.00000000"`
	Source          string                   `json:"source" example:"MANUAL"`
	Entries         []EntryResponse          `json:"entries,omitempty"`
	Attachments     []coremodel.FileResponse `json:"attachments,omitempty"`
	AuditMeta       base.AuditMeta           `json:"auditMeta"`
}
