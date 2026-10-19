// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"github.com/kilip/omed/api/internal/base"
)

// Standard financial account classifications.
const (
	AccountTypeAsset     = "ASSET"
	AccountTypeLiability = "LIABILITY"
	AccountTypeEquity    = "EQUITY"
	AccountTypeRevenue   = "REVENUE"
	AccountTypeExpense   = "EXPENSE"
)

// Normal balance indicators.
const (
	NormalBalanceDebit  = "DEBIT"
	NormalBalanceCredit = "CREDIT"
)

// ValidAccountTypes lists all valid account classifications.
var ValidAccountTypes = map[string]bool{
	AccountTypeAsset:     true,
	AccountTypeLiability: true,
	AccountTypeEquity:    true,
	AccountTypeRevenue:   true,
	AccountTypeExpense:   true,
}

// ValidNormalBalances lists all valid normal balance types.
var ValidNormalBalances = map[string]bool{
	NormalBalanceDebit:  true,
	NormalBalanceCredit: true,
}

// DefaultNormalBalance returns the standard accounting normal balance for an account type.
func DefaultNormalBalance(accountType string) string {
	switch accountType {
	case AccountTypeAsset, AccountTypeExpense:
		return NormalBalanceDebit
	case AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue:
		return NormalBalanceCredit
	default:
		return NormalBalanceDebit
	}
}

// Account represents a Chart of Accounts entity in the domain.
type Account struct {
	ID            int64          `json:"id"`
	WorkspaceID   int64          `json:"workspaceId"`
	Code          string         `json:"code"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	NormalBalance string         `json:"normalBalance"`
	Currency      string         `json:"currency,omitempty"`
	ParentID      int64          `json:"parentId,omitempty"`
	Description   string         `json:"description,omitempty"`
	IsActive      bool           `json:"isActive"`
	IsPostable    bool           `json:"isPostable"`
	AuditMeta     base.AuditMeta `json:"auditMeta"`
}
