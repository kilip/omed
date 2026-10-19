// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Standard transaction source types.
const (
	TransactionSourceManual = "MANUAL"
	TransactionSourceImport = "IMPORT"
	TransactionSourceSystem = "SYSTEM"
)

// ValidTransactionSources lists all valid transaction sources.
var ValidTransactionSources = map[string]bool{
	TransactionSourceManual: true,
	TransactionSourceImport: true,
	TransactionSourceSystem: true,
}

// Transaction represents a financial business event in the domain.
type Transaction struct {
	ID              int64          `json:"id"`
	WorkspaceID     int64          `json:"workspaceId"`
	Number          string         `json:"number"`
	TransactionDate time.Time      `json:"transactionDate"`
	Description     string         `json:"description,omitempty"`
	Reference       string         `json:"reference,omitempty"`
	Currency        string         `json:"currency"`
	ExchangeRate    float64        `json:"exchangeRate"`
	Source          string         `json:"source"`
	Entries         []Entry        `json:"entries,omitempty"`
	AuditMeta       base.AuditMeta `json:"auditMeta"`
}
