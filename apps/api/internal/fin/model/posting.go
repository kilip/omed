// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/kilip/omed/api/internal/base"
)

// CreatePostingRequest defines the payload required to create a posting line.
type CreatePostingRequest struct {
	AccountID   int64    `json:"accountId" validate:"required"`
	Direction   string   `json:"direction" validate:"required,oneof=DEBIT CREDIT" example:"DEBIT"`
	Amount      float64  `json:"amount" validate:"min=0" example:"100000"`
	BaseAmount  *float64 `json:"baseAmount,omitempty"`
	Description string   `json:"description,omitempty"`
}

// PostingResponse represents posting line metadata returned by API endpoints.
type PostingResponse struct {
	ID                 int64          `json:"id"`
	WorkspaceID        int64          `json:"workspaceId"`
	EntryID            int64          `json:"entryId"`
	AccountID          int64          `json:"accountId"`
	Direction          string         `json:"direction" example:"DEBIT"`
	Amount             float64        `json:"amount" example:"100000"`
	BaseAmount         float64        `json:"baseAmount" example:"100000"`
	Position           int            `json:"position" example:"0"`
	Description        string         `json:"description,omitempty"`
	RunningBalance     float64        `json:"runningBalance,omitempty" example:"100000"`
	RunningBaseBalance float64        `json:"runningBaseBalance,omitempty" example:"100000"`
	Balance            float64        `json:"balance,omitempty" example:"100000"`
	BaseBalance        float64        `json:"baseBalance,omitempty" example:"100000"`
	AuditMeta          base.AuditMeta `json:"auditMeta"`
}

// LedgerEntryResponse is an alias for PostingResponse representing a ledger line.
type LedgerEntryResponse = PostingResponse

// AccountLedgerResponse represents an account ledger with postings and running balances.
type AccountLedgerResponse struct {
	AccountID     int64             `json:"accountId"`
	NormalBalance string            `json:"normalBalance" example:"DEBIT"`
	Balance       float64           `json:"balance" example:"1000"`
	BaseBalance   float64           `json:"baseBalance" example:"15000000"`
	Total         int64             `json:"total"`
	Page          int               `json:"page"`
	Limit         int               `json:"limit"`
	Postings      []PostingResponse `json:"postings"`
}
