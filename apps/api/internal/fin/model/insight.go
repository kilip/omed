package model

import (
	"github.com/shopspring/decimal"
)

// PeriodOverviewRequest defines parameters for the period overview insight tool.
type PeriodOverviewRequest struct {
	WorkspaceID     int64  `json:"workspace_id" jsonschema:"description=Workspace ID to query"`
	FromMonth       string `json:"from_month,omitempty" jsonschema:"description=Start month in YYYY-MM format (defaults to current month)"`
	ToMonth         string `json:"to_month,omitempty" jsonschema:"description=End month in YYYY-MM format (defaults to current month)"`
	ComparePrevious *bool  `json:"compare_previous,omitempty" jsonschema:"description=Whether to compare with the previous period of the same length (defaults to true)"`
}

// GetWorkspaceID returns the workspace ID for workspace scoping.
func (r PeriodOverviewRequest) GetWorkspaceID() int64 {
	return r.WorkspaceID
}

// MonthTotals represents revenue, expense, and net totals for a single month.
type MonthTotals struct {
	Month   string          `json:"month"`
	Revenue decimal.Decimal `json:"revenue"`
	Expense decimal.Decimal `json:"expense"`
	Net     decimal.Decimal `json:"net"`
}

// TotalsDelta represents the difference between current and previous period totals.
type TotalsDelta struct {
	Revenue decimal.Decimal `json:"revenue"`
	Expense decimal.Decimal `json:"expense"`
	Net     decimal.Decimal `json:"net"`
}

// PeriodComparison represents totals and monthly figures for a comparative period.
type PeriodComparison struct {
	From    string          `json:"from"`
	To      string          `json:"to"`
	Revenue decimal.Decimal `json:"revenue"`
	Expense decimal.Decimal `json:"expense"`
	Net     decimal.Decimal `json:"net"`
	Monthly []MonthTotals   `json:"monthly"`
	Delta   TotalsDelta     `json:"delta"`
}

// PeriodOverviewResponse represents the financial overview for a requested period.
type PeriodOverviewResponse struct {
	BaseCurrency string            `json:"base_currency"`
	From         string            `json:"from"`
	To           string            `json:"to"`
	Revenue      decimal.Decimal   `json:"revenue"`
	Expense      decimal.Decimal   `json:"expense"`
	Net          decimal.Decimal   `json:"net"`
	Monthly      []MonthTotals     `json:"monthly"`
	Previous     *PeriodComparison `json:"previous,omitempty"`
}

// AccountBalanceRequest defines parameters for the account balance insight tool.
type AccountBalanceRequest struct {
	WorkspaceID int64  `json:"workspace_id" jsonschema:"description=Workspace ID,required"`
	AsOfDate    string `json:"as_of_date,omitempty" jsonschema:"description=As-of date in YYYY-MM-DD format (defaults to current date),pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}$"`
	Account     string `json:"account,omitempty" jsonschema:"description=Optional filter: code prefix (e.g. '11' matches '1100'..'1199') or case-insensitive substring of name"`
}

// GetWorkspaceID returns the workspace ID for authorization and scoping.
func (r AccountBalanceRequest) GetWorkspaceID() int64 {
	return r.WorkspaceID
}

// AccountBalanceItem represents a single account's balance in the insight response.
type AccountBalanceItem struct {
	Code   string          `json:"code"`
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Debit  decimal.Decimal `json:"debit"`
	Credit decimal.Decimal `json:"credit"`
	Net    decimal.Decimal `json:"net"`
}

// BalanceTotals represents the debit and credit totals.
type BalanceTotals struct {
	Debit  decimal.Decimal `json:"debit"`
	Credit decimal.Decimal `json:"credit"`
}

// AccountBalanceResponse represents the account balance insight output.
type AccountBalanceResponse struct {
	AsOfDate     string               `json:"as_of_date"`
	BaseCurrency string               `json:"base_currency"`
	Accounts     []AccountBalanceItem `json:"accounts"`
	Totals       BalanceTotals        `json:"totals"`
	IsBalanced   bool                 `json:"is_balanced"`
	Truncated    bool                 `json:"truncated,omitempty"`
	Hint         string               `json:"hint,omitempty"`
}

// Shrink halves the accounts array to progressively reduce output size when oversize.
func (r *AccountBalanceResponse) Shrink() bool {
	if len(r.Accounts) == 0 {
		return false
	}
	r.Accounts = r.Accounts[:len(r.Accounts)/2]
	r.Truncated = true
	r.Hint = "narrow your account filter"
	return true
}
