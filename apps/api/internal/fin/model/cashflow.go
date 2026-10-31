// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/kilip/omed/api/internal/base"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/shopspring/decimal"
)

// CashFlowItemRequest represents an individual line item in a save cash flow request.
type CashFlowItemRequest struct {
	AccountID   int64           `json:"accountId" validate:"required" example:"40130"`
	Description string          `json:"description,omitempty" example:"Consulting"`
	Quantity    decimal.Decimal `json:"quantity" validate:"required" swaggertype:"string" example:"1"`
	UnitPrice   decimal.Decimal `json:"unitPrice" validate:"required" swaggertype:"string" example:"1000000.00"`
}

// SaveCashFlowRequest defines the payload required to record or update a multi-item cash flow document.
type SaveCashFlowRequest struct {
	PaymentAccountID  int64                 `json:"paymentAccountId" validate:"required" example:"1"`
	TransactionDate   Date                  `json:"transactionDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Reference         string                `json:"reference,omitempty" example:"INV-001"`
	Description       string                `json:"description,omitempty" example:"Office supplies"`
	Currency          string                `json:"currency,omitempty" validate:"omitempty,len=3" example:"IDR"`
	Status            string                `json:"status,omitempty" validate:"omitempty,oneof=DRAFT POSTED" example:"POSTED"`
	ExchangeRate      *decimal.Decimal      `json:"exchangeRate,omitempty" swaggertype:"string" example:"1.00000000"`
	Items             []CashFlowItemRequest `json:"items" validate:"required,min=1,max=100,dive"`
	AttachmentFileIDs []int64               `json:"attachmentFileIds,omitempty"`
}

// SearchCashFlowRequest defines the query/filter parameters for cash flow searching.
type SearchCashFlowRequest struct {
	DateFrom          *Date   `json:"dateFrom,omitempty" swaggertype:"string" example:"2026-01-01"`
	DateTo            *Date   `json:"dateTo,omitempty" swaggertype:"string" example:"2026-01-31"`
	PaymentAccountID  *int64  `json:"paymentAccountId,omitempty" example:"1"`
	CategoryAccountID *int64  `json:"categoryAccountId,omitempty" example:"5"`
	Status            *string `json:"status,omitempty" validate:"omitempty,oneof=DRAFT POSTED" example:"POSTED"`
	Query             *string `json:"query,omitempty" example:"Office"`
	Cursor            *string `json:"cursor,omitempty"`
	Limit             *int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100" example:"20"`
}

// CashFlowAccountSummaryResponse represents a condensed account response in cash flow payloads.
type CashFlowAccountSummaryResponse struct {
	ID   int64  `json:"id" example:"1"`
	Code string `json:"code" example:"1101"`
	Name string `json:"name" example:"Cash"`
	Type string `json:"type" example:"ASSET"`
}

// CashFlowLineResponse represents an individual item line in cash flow response payloads.
type CashFlowLineResponse struct {
	ID          int64                          `json:"id" example:"11"`
	Account     CashFlowAccountSummaryResponse `json:"account"`
	Description string                         `json:"description,omitempty" example:"Konsultasi Improvement"`
	Quantity    decimal.Decimal                `json:"quantity" swaggertype:"string" example:"1"`
	UnitPrice   decimal.Decimal                `json:"unitPrice" swaggertype:"string" example:"1000000.00"`
	Amount      decimal.Decimal                `json:"amount" swaggertype:"string" example:"1000000.00"`
	BaseAmount  decimal.Decimal                `json:"baseAmount" swaggertype:"string" example:"1000000.00"`
}

// CashFlowItemResponse represents a cash flow record returned by the API.
type CashFlowItemResponse struct {
	ID              int64                          `json:"id" example:"1"`
	WorkspaceID     int64                          `json:"workspaceId" example:"1"`
	Number          string                         `json:"number" example:"TRX-2026-000001"`
	TransactionDate Date                           `json:"transactionDate" swaggertype:"string" example:"2026-01-01"`
	Type            string                         `json:"type" example:"EXPENSE"`
	PaymentAccount  CashFlowAccountSummaryResponse `json:"paymentAccount"`
	Items           []CashFlowLineResponse         `json:"items"`
	Amount          decimal.Decimal                `json:"amount" swaggertype:"string" example:"100000.00"`
	BaseAmount      decimal.Decimal                `json:"baseAmount" swaggertype:"string" example:"100000.00"`
	Currency        string                         `json:"currency" example:"IDR"`
	ExchangeRate    decimal.Decimal                `json:"exchangeRate" swaggertype:"string" example:"1.00000000"`
	Description     string                         `json:"description,omitempty" example:"Office supplies"`
	Reference       string                         `json:"reference,omitempty" example:"INV-001"`
	EntryID         int64                          `json:"entryId" example:"1"`
	EntryNumber     string                         `json:"entryNumber" example:"JRN-2026-000001"`
	EntryStatus     string                         `json:"entryStatus" example:"POSTED"`
	Attachments     []coremodel.FileResponse       `json:"attachments,omitempty"`
	AuditMeta       base.AuditMeta                 `json:"auditMeta"`
}
