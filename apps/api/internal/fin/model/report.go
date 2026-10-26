// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/shopspring/decimal"
)

// TrialBalanceRequest defines the request payload for trial balance reporting.
type TrialBalanceRequest struct {
	AsOfDate                    Date  `json:"asOfDate" validate:"required" swaggertype:"string" example:"2026-01-31"`
	IncludePerCurrencyBreakdown *bool `json:"includePerCurrencyBreakdown,omitempty" example:"false"`
}

// CurrencyBreakdownItem represents balance details per foreign currency for an account.
type CurrencyBreakdownItem struct {
	Currency   string          `json:"currency" example:"USD"`
	Debit      decimal.Decimal `json:"debit" swaggertype:"string" example:"100.00"`
	Credit     decimal.Decimal `json:"credit" swaggertype:"string" example:"0.00"`
	NetBalance decimal.Decimal `json:"netBalance" swaggertype:"string" example:"100.00"`
	BaseDebit  decimal.Decimal `json:"baseDebit" swaggertype:"string" example:"1600000.00"`
	BaseCredit decimal.Decimal `json:"baseCredit" swaggertype:"string" example:"0.00"`
	BaseNet    decimal.Decimal `json:"baseNet" swaggertype:"string" example:"1600000.00"`
}

// TrialBalanceAccountItem represents an account's balances in base currency on the trial balance.
type TrialBalanceAccountItem struct {
	AccountID   int64                   `json:"accountId"`
	AccountCode string                  `json:"accountCode" example:"1110"`
	AccountName string                  `json:"accountName" example:"Bank BCA IDR"`
	Type        string                  `json:"type" example:"ASSET"`
	Debit       decimal.Decimal         `json:"debit" swaggertype:"string" example:"1000000.00"`
	Credit      decimal.Decimal         `json:"credit" swaggertype:"string" example:"0.00"`
	NetBalance  decimal.Decimal         `json:"netBalance" swaggertype:"string" example:"1000000.00"`
	Currencies  []CurrencyBreakdownItem `json:"currencies,omitempty"`
}

// TrialBalanceResponse represents the result of a trial balance calculation.
type TrialBalanceResponse struct {
	AsOfDate     Date                      `json:"asOfDate" swaggertype:"string" example:"2026-01-31"`
	BaseCurrency string                    `json:"baseCurrency" example:"IDR"`
	Accounts     []TrialBalanceAccountItem `json:"accounts"`
	TotalDebit   decimal.Decimal           `json:"totalDebit" swaggertype:"string" example:"5000000.00"`
	TotalCredit  decimal.Decimal           `json:"totalCredit" swaggertype:"string" example:"5000000.00"`
	IsBalanced   bool                      `json:"isBalanced" example:"true"`
}

// FinancialSummaryTotals represents cumulative balance sheet and income totals.
type FinancialSummaryTotals struct {
	Assets      decimal.Decimal `json:"assets" swaggertype:"string" example:"150000000.00"`
	Liabilities decimal.Decimal `json:"liabilities" swaggertype:"string" example:"35000000.00"`
	Equity      decimal.Decimal `json:"equity" swaggertype:"string" example:"115000000.00"`
	Revenue     decimal.Decimal `json:"revenue" swaggertype:"string" example:"85000000.00"`
	Expense     decimal.Decimal `json:"expense" swaggertype:"string" example:"45000000.00"`
	NetIncome   decimal.Decimal `json:"netIncome" swaggertype:"string" example:"40000000.00"`
}

// MonthlyRevenueExpenseItem represents revenue and expense totals for a specific month.
type MonthlyRevenueExpenseItem struct {
	Month   string          `json:"month" example:"2026-10"`
	Revenue decimal.Decimal `json:"revenue" swaggertype:"string" example:"25000000.00"`
	Expense decimal.Decimal `json:"expense" swaggertype:"string" example:"15000000.00"`
}

// FinancialSummaryResponse represents aggregated totals and monthly revenue vs expense breakdown.
type FinancialSummaryResponse struct {
	BaseCurrency string                      `json:"baseCurrency" example:"IDR"`
	Totals       FinancialSummaryTotals      `json:"totals"`
	Monthly      []MonthlyRevenueExpenseItem `json:"monthly"`
}
