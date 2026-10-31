package model

import (
	"github.com/shopspring/decimal"
)

// PeriodOverviewRequest defines parameters for the period overview insight tool.
type PeriodOverviewRequest struct {
	WorkspaceID     int64  `json:"workspace_id" jsonschema:"Workspace ID to query"`
	FromMonth       string `json:"from_month,omitempty" jsonschema:"Start month in YYYY-MM format (defaults to current month)"`
	ToMonth         string `json:"to_month,omitempty" jsonschema:"End month in YYYY-MM format (defaults to current month)"`
	ComparePrevious *bool  `json:"compare_previous,omitempty" jsonschema:"Whether to compare with the previous period of the same length (defaults to true)"`
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
	WorkspaceID int64  `json:"workspace_id" jsonschema:"Workspace ID"`
	AsOfDate    string `json:"as_of_date,omitempty" jsonschema:"As-of date in YYYY-MM-DD format (defaults to current date)"`
	Account     string `json:"account,omitempty" jsonschema:"Optional filter: code prefix (e.g. '11' matches '1100'..'1199') or case-insensitive substring of name"`
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

// AccountRef represents a minimal account reference with code and name.
type AccountRef struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// FindCashflowRequest defines parameters for the find cashflow insight tool.
type FindCashflowRequest struct {
	WorkspaceID     int64  `json:"workspace_id" jsonschema:"Workspace ID"`
	Type            string `json:"type" jsonschema:"Document type: 'income' or 'expense'"`
	DateFrom        string `json:"date_from,omitempty" jsonschema:"Start date in YYYY-MM-DD format (inclusive)"`
	DateTo          string `json:"date_to,omitempty" jsonschema:"End date in YYYY-MM-DD format (inclusive)"`
	Status          string `json:"status,omitempty" jsonschema:"Document status: 'draft' or 'posted'"`
	Query           string `json:"query,omitempty" jsonschema:"Text search across description and reference"`
	PaymentAccount  string `json:"payment_account,omitempty" jsonschema:"Account code or name substring for payment/cash account"`
	CategoryAccount string `json:"category_account,omitempty" jsonschema:"Account code or name substring for category account"`
	Cursor          string `json:"cursor,omitempty" jsonschema:"Opaque pagination cursor from previous response"`
	Limit           int    `json:"limit,omitempty" jsonschema:"Maximum items to return (1-50, default 20)"`
}

// GetWorkspaceID returns the workspace ID for authorization and scoping.
func (r FindCashflowRequest) GetWorkspaceID() int64 {
	return r.WorkspaceID
}

// CashflowLine represents a single line in a cashflow document.
type CashflowLine struct {
	Account     AccountRef      `json:"account"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Amount      decimal.Decimal `json:"amount"`
}

// CashflowDoc represents a sanitized, public cashflow document for insights.
type CashflowDoc struct {
	ID             int64           `json:"id"`
	Number         string          `json:"number"`
	Date           string          `json:"date"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	PaymentAccount AccountRef      `json:"payment_account"`
	Description    string          `json:"description"`
	Reference      string          `json:"reference"`
	Amount         decimal.Decimal `json:"amount"`
	Currency       string          `json:"currency"`
	BaseAmount     decimal.Decimal `json:"base_amount"`
	Lines          []CashflowLine  `json:"lines"`
}

// FindCashflowResponse is the output of the find cashflow insight tool.
type FindCashflowResponse struct {
	Items      []CashflowDoc `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
	Truncated  bool          `json:"truncated,omitempty"`
	Hint       string        `json:"hint,omitempty"`
}

// Shrink halves the items array to progressively reduce output size when oversize.
func (r *FindCashflowResponse) Shrink() bool {
	if len(r.Items) == 0 {
		return false
	}
	r.Items = r.Items[:len(r.Items)/2]
	r.Truncated = true
	r.Hint = "narrow your search or reduce limit"
	return true
}
