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
