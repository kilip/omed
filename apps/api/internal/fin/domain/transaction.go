// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
)

// Standard transaction source types and entity type constants.
const (
	TransactionSourceManual = "MANUAL"
	TransactionSourceImport = "IMPORT"
	TransactionSourceSystem = "SYSTEM"

	// TransactionEntityType is the entityType stored on core_file records linked to a transaction.
	TransactionEntityType = "fin:transaction"
	// DefaultMaxTransactionAttachmentSize is the default maximum attachment size in bytes (10 MB).
	DefaultMaxTransactionAttachmentSize int64 = 10 * 1024 * 1024
)

// DefaultAllowedAttachmentTypes lists the default allowed file extensions for transaction attachments.
var DefaultAllowedAttachmentTypes = []string{
	".pdf",
	".png",
	".jpg",
	".jpeg",
	".webp",
	".csv",
	".xls",
	".xlsx",
}

// ValidTransactionSources lists all valid transaction sources.
var ValidTransactionSources = map[string]bool{
	TransactionSourceManual: true,
	TransactionSourceImport: true,
	TransactionSourceSystem: true,
}

// Transaction represents a financial business event in the domain.
type Transaction struct {
	ID              int64             `json:"id"`
	WorkspaceID     int64             `json:"workspaceId"`
	Number          string            `json:"number"`
	TransactionDate time.Time         `json:"transactionDate"`
	Description     string            `json:"description,omitempty"`
	Reference       string            `json:"reference,omitempty"`
	Currency        string            `json:"currency"`
	ExchangeRate    float64           `json:"exchangeRate"`
	Source          string            `json:"source"`
	Entries         []Entry           `json:"entries,omitempty"`
	Attachments     []coredomain.File `json:"attachments,omitempty"`
	AuditMeta       base.AuditMeta    `json:"auditMeta"`
}
