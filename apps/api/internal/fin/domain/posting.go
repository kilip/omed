// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"github.com/kilip/omed/api/internal/base"
)

// Direction constants for postings.
const (
	PostingDirectionDebit  = "DEBIT"
	PostingDirectionCredit = "CREDIT"
)

// ValidPostingDirections lists all valid posting directions.
var ValidPostingDirections = map[string]bool{
	PostingDirectionDebit:  true,
	PostingDirectionCredit: true,
}

// Posting represents a debit or credit entry line in the domain.
type Posting struct {
	ID          int64          `json:"id"`
	WorkspaceID int64          `json:"workspaceId"`
	EntryID     int64          `json:"entryId"`
	AccountID   int64          `json:"accountId"`
	Direction   string         `json:"direction"`
	Amount      float64        `json:"amount"`
	BaseAmount  float64        `json:"baseAmount"`
	Position    int            `json:"position"`
	Description        string         `json:"description,omitempty"`
	RunningBalance     float64        `json:"runningBalance,omitempty"`
	RunningBaseBalance float64        `json:"runningBaseBalance,omitempty"`
	Balance            float64        `json:"balance,omitempty"`
	BaseBalance        float64        `json:"baseBalance,omitempty"`
	AuditMeta          base.AuditMeta `json:"auditMeta"`
}
