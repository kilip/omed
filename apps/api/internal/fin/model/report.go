// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

// TrialBalanceRequest defines the request payload for trial balance reporting.
type TrialBalanceRequest struct {
	AsOfDate                    Date  `json:"asOfDate" validate:"required" swaggertype:"string" example:"2026-01-31"`
	IncludePerCurrencyBreakdown *bool `json:"includePerCurrencyBreakdown,omitempty" example:"false"`
}

// CurrencyBreakdownItem represents balance details per foreign currency for an account.
type CurrencyBreakdownItem struct {
	Currency   string  `json:"currency" example:"USD"`
	Debit      float64 `json:"debit" example:"100.00"`
	Credit     float64 `json:"credit" example:"0.00"`
	NetBalance float64 `json:"netBalance" example:"100.00"`
	BaseDebit  float64 `json:"baseDebit" example:"1600000.00"`
	BaseCredit float64 `json:"baseCredit" example:"0.00"`
	BaseNet    float64 `json:"baseNet" example:"1600000.00"`
}

// TrialBalanceAccountItem represents an account's balances in base currency on the trial balance.
type TrialBalanceAccountItem struct {
	AccountID   int64                   `json:"accountId"`
	AccountCode string                  `json:"accountCode" example:"1110"`
	AccountName string                  `json:"accountName" example:"Bank BCA IDR"`
	Type        string                  `json:"type" example:"ASSET"`
	Debit       float64                 `json:"debit" example:"1000000"`
	Credit      float64                 `json:"credit" example:"0"`
	NetBalance  float64                 `json:"netBalance" example:"1000000"`
	Currencies  []CurrencyBreakdownItem `json:"currencies,omitempty"`
}

// TrialBalanceResponse represents the result of a trial balance calculation.
type TrialBalanceResponse struct {
	AsOfDate     Date                      `json:"asOfDate" swaggertype:"string" example:"2026-01-31"`
	BaseCurrency string                    `json:"baseCurrency" example:"IDR"`
	Accounts     []TrialBalanceAccountItem `json:"accounts"`
	TotalDebit   float64                   `json:"totalDebit" example:"5000000"`
	TotalCredit  float64                   `json:"totalCredit" example:"5000000"`
	IsBalanced   bool                      `json:"isBalanced" example:"true"`
}

// FinancialSummaryTotals represents cumulative balance sheet and income totals.
type FinancialSummaryTotals struct {
	Assets      float64 `json:"assets" example:"150000000"`
	Liabilities float64 `json:"liabilities" example:"35000000"`
	Equity      float64 `json:"equity" example:"115000000"`
	Revenue     float64 `json:"revenue" example:"85000000"`
	Expense     float64 `json:"expense" example:"45000000"`
	NetIncome   float64 `json:"netIncome" example:"40000000"`
}

// MonthlyRevenueExpenseItem represents revenue and expense totals for a specific month.
type MonthlyRevenueExpenseItem struct {
	Month   string  `json:"month" example:"2026-10"`
	Revenue float64 `json:"revenue" example:"25000000"`
	Expense float64 `json:"expense" example:"15000000"`
}

// FinancialSummaryResponse represents aggregated totals and monthly revenue vs expense breakdown.
type FinancialSummaryResponse struct {
	BaseCurrency string                      `json:"baseCurrency" example:"IDR"`
	Totals       FinancialSummaryTotals      `json:"totals"`
	Monthly      []MonthlyRevenueExpenseItem `json:"monthly"`
}
