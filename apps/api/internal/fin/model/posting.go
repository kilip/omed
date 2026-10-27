// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/kilip/omed/api/internal/base"
	"github.com/shopspring/decimal"
)

// CreatePostingRequest defines the payload required to create a posting line.
type CreatePostingRequest struct {
	AccountID   int64            `json:"accountId" validate:"required"`
	Direction   string           `json:"direction" validate:"required,oneof=DEBIT CREDIT" example:"DEBIT"`
	Amount      decimal.Decimal  `json:"amount" swaggertype:"string" example:"100000.00"`
	BaseAmount  *decimal.Decimal `json:"baseAmount,omitempty" swaggertype:"string" example:"100000.00"`
	Quantity    *decimal.Decimal `json:"quantity,omitempty" swaggertype:"string" example:"4"`
	UnitPrice   *decimal.Decimal `json:"unitPrice,omitempty" swaggertype:"string" example:"22000.00"`
	Description string           `json:"description,omitempty"`
}

// PostingResponse represents posting line metadata returned by API endpoints.
type PostingResponse struct {
	ID                 int64            `json:"id"`
	WorkspaceID        int64            `json:"workspaceId"`
	EntryID            int64            `json:"entryId"`
	AccountID          int64            `json:"accountId"`
	Direction          string           `json:"direction" example:"DEBIT"`
	Amount             decimal.Decimal  `json:"amount" swaggertype:"string" example:"100000.00"`
	BaseAmount         decimal.Decimal  `json:"baseAmount" swaggertype:"string" example:"100000.00"`
	Quantity           *decimal.Decimal `json:"quantity,omitempty" swaggertype:"string" example:"4"`
	UnitPrice          *decimal.Decimal `json:"unitPrice,omitempty" swaggertype:"string" example:"22000.00"`
	Position           int              `json:"position" example:"0"`
	Description        string           `json:"description,omitempty"`
	RunningBalance     decimal.Decimal  `json:"runningBalance,omitempty" swaggertype:"string" example:"100000.00"`
	RunningBaseBalance decimal.Decimal  `json:"runningBaseBalance,omitempty" swaggertype:"string" example:"100000.00"`
	Balance            decimal.Decimal  `json:"balance,omitempty" swaggertype:"string" example:"100000.00"`
	BaseBalance        decimal.Decimal  `json:"baseBalance,omitempty" swaggertype:"string" example:"100000.00"`
	AuditMeta          base.AuditMeta   `json:"auditMeta"`
}

// LedgerEntryResponse is an alias for PostingResponse representing a ledger line.
type LedgerEntryResponse = PostingResponse

// AccountLedgerResponse represents an account ledger with postings and running balances.
type AccountLedgerResponse struct {
	AccountID     int64             `json:"accountId"`
	NormalBalance string            `json:"normalBalance" example:"DEBIT"`
	Balance       decimal.Decimal   `json:"balance" swaggertype:"string" example:"1000.00"`
	BaseBalance   decimal.Decimal   `json:"baseBalance" swaggertype:"string" example:"15000000.00"`
	Total         int64             `json:"total"`
	Page          int               `json:"page"`
	Limit         int               `json:"limit"`
	Postings      []PostingResponse `json:"postings"`
}
