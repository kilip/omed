// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	"github.com/shopspring/decimal"
)

// CashFlowType defines the classification of a quick-entry cash flow (INCOME or EXPENSE).
type CashFlowType = string

const (
	CashFlowTypeIncome  CashFlowType = "INCOME"
	CashFlowTypeExpense CashFlowType = "EXPENSE"
)

// CashFlowAccountSummary holds a condensed representation of an account for cash flow items.
type CashFlowAccountSummary struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// CashFlowItem represents a flattened, user-facing cash flow event (Income or Expense)
// backed by an underlying financial transaction and posted journal entry.
type CashFlowItem struct {
	ID              int64                  `json:"id"`
	WorkspaceID     int64                  `json:"workspaceId"`
	Number          string                 `json:"number"`
	TransactionDate time.Time              `json:"transactionDate"`
	Type            string                 `json:"type"`
	PaymentAccount  CashFlowAccountSummary `json:"paymentAccount"`
	CategoryAccount CashFlowAccountSummary `json:"categoryAccount"`
	Amount          decimal.Decimal        `json:"amount"`
	BaseAmount      decimal.Decimal        `json:"baseAmount"`
	Currency        string                 `json:"currency"`
	ExchangeRate    decimal.Decimal        `json:"exchangeRate"`
	Description     string                 `json:"description,omitempty"`
	Reference       string                 `json:"reference,omitempty"`
	EntryID         int64                  `json:"entryId"`
	EntryNumber     string                 `json:"entryNumber"`
	EntryStatus     string                 `json:"entryStatus"`
	Attachments     []coredomain.File      `json:"attachments,omitempty"`
	AuditMeta       base.AuditMeta         `json:"auditMeta"`
}
