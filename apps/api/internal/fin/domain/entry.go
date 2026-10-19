// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Standard journal entry statuses.
const (
	EntryStatusDraft  = "DRAFT"
	EntryStatusPosted = "POSTED"
	EntryStatusVoid   = "VOID"
)

// ValidEntryStatuses lists all valid journal entry statuses.
var ValidEntryStatuses = map[string]bool{
	EntryStatusDraft:  true,
	EntryStatusPosted: true,
	EntryStatusVoid:   true,
}

// Entry represents a Journal Entry entity in the domain.
type Entry struct {
	ID           int64          `json:"id"`
	WorkspaceID  int64          `json:"workspaceId"`
	TransactionID int64         `json:"transactionId"`
	PeriodID     int64          `json:"periodId"`
	Number       string         `json:"number"`
	EntryDate    time.Time      `json:"entryDate"`
	Description  string         `json:"description,omitempty"`
	Status       string         `json:"status"`
	PostedAt     *time.Time     `json:"postedAt,omitempty"`
	PostedBy     *int64         `json:"postedBy,omitempty"`
	ReversalOfID *int64         `json:"reversalOfId,omitempty"`
	Postings     []Posting      `json:"postings,omitempty"`
	AuditMeta    base.AuditMeta `json:"auditMeta"`
}
