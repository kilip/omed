package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/shopspring/decimal"
)

// CurrencyOption represents a currency available in the workspace.
type CurrencyOption struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// PeriodOption represents an accounting period in the workspace.
type PeriodOption struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	FiscalYear int       `json:"fiscalYear"`
	Status     string    `json:"status"`
}

// AccountCurrencyBalance represents foreign currency debit/credit breakdown for an account.
type AccountCurrencyBalance struct {
	Currency   string          `json:"currency"`
	Debit      decimal.Decimal `json:"debit"`
	Credit     decimal.Decimal `json:"credit"`
	NetBalance decimal.Decimal `json:"netBalance"`
	BaseDebit  decimal.Decimal `json:"baseDebit"`
	BaseCredit decimal.Decimal `json:"baseCredit"`
	BaseNet    decimal.Decimal `json:"baseNet"`
}

// AccountBalanceItem represents a single account's balance in base currency.
type AccountBalanceItem struct {
	AccountID   int64                    `json:"accountId,omitempty"`
	Code        string                   `json:"code"`
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Debit       decimal.Decimal          `json:"debit"`
	Credit      decimal.Decimal          `json:"credit"`
	NetBalance  decimal.Decimal          `json:"netBalance"`
	Currencies  []AccountCurrencyBalance `json:"currencies,omitempty"`
}

// AccountBalanceQuery defines filter and as-of date parameters for querying account balances.
type AccountBalanceQuery struct {
	AsOfDate                    string `json:"asOfDate,omitempty"`
	Account                     string `json:"account,omitempty"`
	Type                        string `json:"type,omitempty"`
	IncludePerCurrencyBreakdown bool   `json:"includePerCurrencyBreakdown,omitempty"`
}

// AccountBalanceResult holds account balances and cumulative aggregates.
type AccountBalanceResult struct {
	AsOfDate        string               `json:"asOfDate"`
	BaseCurrency    string               `json:"baseCurrency"`
	Accounts        []AccountBalanceItem `json:"accounts"`
	TotalDebit      decimal.Decimal      `json:"totalDebit"`
	TotalCredit     decimal.Decimal      `json:"totalCredit"`
	MatchedNetTotal decimal.Decimal      `json:"matchedNetTotal"`
	IsBalanced      bool                 `json:"isBalanced"`
	Truncated       bool                 `json:"truncated,omitempty"`
	Hint            string               `json:"hint,omitempty"`
}

// FinancialSummaryTotals represents cumulative balance sheet and income totals.
type FinancialSummaryTotals struct {
	Assets      decimal.Decimal `json:"assets"`
	Liabilities decimal.Decimal `json:"liabilities"`
	Equity      decimal.Decimal `json:"equity"`
	Revenue     decimal.Decimal `json:"revenue"`
	Expense     decimal.Decimal `json:"expense"`
	NetIncome   decimal.Decimal `json:"netIncome"`
}

// MonthlyRevenueExpense represents revenue and expense totals for a specific month.
type MonthlyRevenueExpense struct {
	Month   string          `json:"month"`
	Revenue decimal.Decimal `json:"revenue"`
	Expense decimal.Decimal `json:"expense"`
	Net     decimal.Decimal `json:"net"`
}

// FinancialSummaryQuery defines period parameters for summary and trend queries.
type FinancialSummaryQuery struct {
	FromMonth       string `json:"fromMonth,omitempty"`
	ToMonth         string `json:"toMonth,omitempty"`
	ComparePrevious bool   `json:"comparePrevious,omitempty"`
}

// FinancialSummaryResult holds overall financial totals and monthly trend breakdown.
type FinancialSummaryResult struct {
	BaseCurrency string                  `json:"baseCurrency"`
	Totals       FinancialSummaryTotals  `json:"totals"`
	Monthly      []MonthlyRevenueExpense `json:"monthly"`
}

// ProfitLossQuery defines date range parameters for Profit & Loss queries.
type ProfitLossQuery struct {
	DateFrom string `json:"dateFrom,omitempty"`
	DateTo   string `json:"dateTo,omitempty"`
}

// ProfitLossCategoryItem represents an account's aggregated net revenue or expense in base currency.
type ProfitLossCategoryItem struct {
	AccountID   int64           `json:"accountId"`
	AccountCode string          `json:"accountCode"`
	AccountName string          `json:"accountName"`
	Type        string          `json:"type"`
	Amount      decimal.Decimal `json:"amount"`
}

// ProfitLossResult represents the category-level Profit & Loss breakdown for a date range.
type ProfitLossResult struct {
	DateFrom     string                   `json:"dateFrom"`
	DateTo       string                   `json:"dateTo"`
	BaseCurrency string                   `json:"baseCurrency"`
	Revenues     []ProfitLossCategoryItem `json:"revenues"`
	Expenses     []ProfitLossCategoryItem `json:"expenses"`
	TotalRevenue decimal.Decimal          `json:"totalRevenue"`
	TotalExpense decimal.Decimal          `json:"totalExpense"`
	NetProfit    decimal.Decimal          `json:"netProfit"`
}

// CashflowLineItem represents a line item within a cash flow document.
type CashflowLineItem struct {
	AccountCode string          `json:"accountCode"`
	AccountName string          `json:"accountName"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Amount      decimal.Decimal `json:"amount"`
}

// CashflowDocItem represents a cash flow document in search results.
type CashflowDocItem struct {
	ID                 int64              `json:"id"`
	Number             string             `json:"number"`
	Date               string             `json:"date"`
	Type               string             `json:"type"`
	Status             string             `json:"status"`
	PaymentAccountCode string             `json:"paymentAccountCode"`
	PaymentAccountName string             `json:"paymentAccountName"`
	Description        string             `json:"description"`
	Reference          string             `json:"reference"`
	Amount             decimal.Decimal    `json:"amount"`
	Currency           string             `json:"currency"`
	BaseAmount         decimal.Decimal    `json:"baseAmount"`
	Lines              []CashflowLineItem `json:"lines,omitempty"`
}

// CashflowSearchQuery defines filters for searching cash flow documents.
type CashflowSearchQuery struct {
	Type            string `json:"type,omitempty"`
	DateFrom        string `json:"dateFrom,omitempty"`
	DateTo          string `json:"dateTo,omitempty"`
	Status          string `json:"status,omitempty"`
	Query           string `json:"query,omitempty"`
	PaymentAccount  string `json:"paymentAccount,omitempty"`
	CategoryAccount string `json:"categoryAccount,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

// CashflowSearchResult holds search results and server-aggregated totals for cash flow documents.
type CashflowSearchResult struct {
	Items                 []CashflowDocItem `json:"items"`
	ReturnedBaseAmountSum decimal.Decimal   `json:"returnedBaseAmountSum"`
	NextCursor            string            `json:"nextCursor,omitempty"`
	Truncated             bool              `json:"truncated,omitempty"`
	Hint                  string            `json:"hint,omitempty"`
}

// AccountLedgerQuery defines account and pagination parameters for ledger (mutasi/buku besar) queries.
type AccountLedgerQuery struct {
	Account string `json:"account"`
	Page    int    `json:"page,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// AccountLedgerItem represents a single posting entry line in an account's ledger.
type AccountLedgerItem struct {
	PostingID          int64           `json:"postingId"`
	EntryID            int64           `json:"entryId"`
	EntryNumber        string          `json:"entryNumber,omitempty"`
	EntryDate          string          `json:"entryDate,omitempty"`
	Direction          string          `json:"direction"`
	Amount             decimal.Decimal `json:"amount"`
	BaseAmount         decimal.Decimal `json:"baseAmount"`
	Currency           string          `json:"currency,omitempty"`
	Description        string          `json:"description,omitempty"`
	RunningBalance     decimal.Decimal `json:"runningBalance"`
	RunningBaseBalance decimal.Decimal `json:"runningBaseBalance"`
}

// AccountLedgerResult holds paginated ledger postings for an account.
type AccountLedgerResult struct {
	AccountID          int64               `json:"accountId"`
	AccountCode        string              `json:"accountCode"`
	AccountName        string              `json:"accountName"`
	Type               string              `json:"type"`
	Currency           string              `json:"currency"`
	BaseCurrency       string              `json:"baseCurrency"`
	Postings           []AccountLedgerItem `json:"postings"`
	TotalCount         int64               `json:"totalCount"`
	Page               int                 `json:"page"`
	Limit              int                 `json:"limit"`
	AmbiguousMatches   []AccountOption     `json:"ambiguousMatches,omitempty"`
}

// TransactionSearchQuery defines filter and pagination parameters for searching general journal transactions.
type TransactionSearchQuery struct {
	DateFrom  string `json:"dateFrom,omitempty"`
	DateTo    string `json:"dateTo,omitempty"`
	Reference string `json:"reference,omitempty"`
	Source    string `json:"source,omitempty"`
	Query     string `json:"query,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// TransactionItem represents a financial transaction header with optional entry count.
type TransactionItem struct {
	ID              int64           `json:"id"`
	Number          string          `json:"number"`
	TransactionDate string          `json:"transactionDate"`
	Description     string          `json:"description,omitempty"`
	Reference       string          `json:"reference,omitempty"`
	Currency        string          `json:"currency"`
	ExchangeRate    decimal.Decimal `json:"exchangeRate"`
	Source          string          `json:"source"`
	EntryCount      int             `json:"entryCount"`
}

// TransactionSearchResult holds matching transactions and pagination cursor.
type TransactionSearchResult struct {
	Items      []TransactionItem `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
}

// ExchangeRateQuery defines lookup and search parameters for foreign exchange rates.
type ExchangeRateQuery struct {
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency,omitempty"`
	Date         string `json:"date,omitempty"`
}

// ExchangeRateItem represents an exchange rate record between two currencies.
type ExchangeRateItem struct {
	ID            int64           `json:"id"`
	FromCurrency  string          `json:"fromCurrency"`
	ToCurrency    string          `json:"toCurrency"`
	Rate          decimal.Decimal `json:"rate"`
	EffectiveDate string          `json:"effectiveDate"`
	Source        string          `json:"source,omitempty"`
}

// ExchangeRateResult holds exchange rate lookup or search results.
type ExchangeRateResult struct {
	Rate         decimal.Decimal    `json:"rate,omitempty"`
	FromCurrency string             `json:"fromCurrency"`
	ToCurrency   string             `json:"toCurrency"`
	Date         string             `json:"date,omitempty"`
	Rates        []ExchangeRateItem `json:"rates,omitempty"`
}

// FinanceDraftContext defines the operations required by agent services to read reference
// finance data, execute financial queries, and persist confirmed draft proposals without posting them.
type FinanceDraftContext interface {
	ListAccounts(ctx context.Context) ([]AccountOption, error)
	ListCurrencies(ctx context.Context) ([]CurrencyOption, error)
	ListPeriods(ctx context.Context) ([]PeriodOption, error)
	SaveDraft(ctx context.Context, proposal model.DraftProposal) (int64, error)

	GetAccountBalances(ctx context.Context, req AccountBalanceQuery) (AccountBalanceResult, error)
	GetFinancialSummary(ctx context.Context, req FinancialSummaryQuery) (FinancialSummaryResult, error)
	GetProfitLossBreakdown(ctx context.Context, req ProfitLossQuery) (ProfitLossResult, error)
	SearchCashflows(ctx context.Context, req CashflowSearchQuery) (CashflowSearchResult, error)
	GetAccountLedger(ctx context.Context, req AccountLedgerQuery) (AccountLedgerResult, error)
	SearchTransactions(ctx context.Context, req TransactionSearchQuery) (TransactionSearchResult, error)
	LookupExchangeRate(ctx context.Context, req ExchangeRateQuery) (ExchangeRateResult, error)
}
