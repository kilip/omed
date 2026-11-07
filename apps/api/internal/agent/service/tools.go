package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/shopspring/decimal"
)

// Tool names for Talk-to-Agent chat assistant.
const (
	ToolListAccounts          = "list_accounts"
	ToolListCurrencies        = "list_currencies"
	ToolListPeriods           = "list_periods"
	ToolProposeDraft          = "propose_draft"
	ToolGetAccountBalances    = "get_account_balances"
	ToolGetFinancialSummary   = "get_financial_summary"
	ToolGetProfitLossBreakdown = "get_profit_loss_breakdown"
	ToolSearchCashflows       = "search_cashflows"
	ToolGetAccountLedger      = "get_account_ledger"
	ToolSearchTransactions    = "search_transactions"
	ToolLookupExchangeRate    = "lookup_exchange_rate"
)

var listAccountsSchema = json.RawMessage(`{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}`)

var listCurrenciesSchema = json.RawMessage(`{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}`)

var listPeriodsSchema = json.RawMessage(`{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}`)

var proposeDraftSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "kind": {
      "type": "string",
      "enum": ["expense", "income", "journal"],
      "description": "Kind of draft proposal: expense, income, or journal"
    },
    "transactionDate": {
      "type": "string",
      "description": "Transaction date in YYYY-MM-DD format"
    },
    "description": {
      "type": "string",
      "description": "Description or memo for the transaction"
    },
    "reference": {
      "type": "string",
      "description": "Optional reference number or invoice/receipt code"
    },
    "currency": {
      "type": "string",
      "description": "Currency code (e.g. IDR, USD)"
    },
    "exchangeRate": {
      "type": "string",
      "description": "Optional exchange rate to base currency if different from workspace currency"
    },
    "paymentAccountId": {
      "type": "integer",
      "description": "Account ID for payment or deposit (required for expense and income drafts)"
    },
    "periodId": {
      "type": "integer",
      "description": "Optional open accounting period ID (e.g. for journal drafts)"
    },
    "totalAmount": {
      "type": "string",
      "description": "Total amount of the transaction as a decimal string (e.g. '70000.00')"
    },
    "items": {
      "type": "array",
      "description": "Line items for expense or income drafts (at least 1 item required)",
      "items": {
        "type": "object",
        "properties": {
          "accountId": {
            "type": "integer",
            "description": "Category or expense/income account ID"
          },
          "description": {
            "type": "string",
            "description": "Line item description"
          },
          "quantity": {
            "type": "string",
            "description": "Item quantity (> 0)"
          },
          "unitPrice": {
            "type": "string",
            "description": "Unit price (>= 0)"
          },
          "amount": {
            "type": "string",
            "description": "Line amount (quantity * unit price)"
          }
        },
        "required": ["accountId", "quantity", "unitPrice"]
      }
    },
    "entries": {
      "type": "array",
      "description": "Journal entries for journal drafts (at least 2 entries; balanced Dr = Cr)",
      "items": {
        "type": "object",
        "properties": {
          "accountId": {
            "type": "integer",
            "description": "Account ID"
          },
          "direction": {
            "type": "string",
            "enum": ["DEBIT", "CREDIT"],
            "description": "Entry direction: DEBIT or CREDIT"
          },
          "amount": {
            "type": "string",
            "description": "Entry amount (> 0)"
          },
          "description": {
            "type": "string",
            "description": "Line description"
          }
        },
        "required": ["accountId", "direction", "amount"]
      }
    }
  },
  "required": ["kind", "transactionDate", "description", "currency", "totalAmount"]
}`)

var getAccountBalancesSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "asOfDate": {
      "type": "string",
      "description": "Optional as-of date in YYYY-MM-DD format (defaults to current date)"
    },
    "account": {
      "type": "string",
      "description": "Optional account code prefix (e.g. '11', '51') or name substring (e.g. 'Kas', 'Bank')"
    },
    "type": {
      "type": "string",
      "enum": ["ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"],
      "description": "Optional account type filter"
    },
    "includePerCurrencyBreakdown": {
      "type": "boolean",
      "description": "Optional flag to include multi-currency balance breakdown"
    }
  },
  "additionalProperties": false
}`)

var getFinancialSummarySchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "fromMonth": {
      "type": "string",
      "description": "Optional starting month in YYYY-MM format"
    },
    "toMonth": {
      "type": "string",
      "description": "Optional ending month in YYYY-MM format"
    },
    "comparePrevious": {
      "type": "boolean",
      "description": "Optional flag to compare with previous period"
    }
  },
  "additionalProperties": false
}`)

var getProfitLossBreakdownSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "dateFrom": {
      "type": "string",
      "description": "Optional start date in YYYY-MM-DD format (defaults to first day of current month)"
    },
    "dateTo": {
      "type": "string",
      "description": "Optional end date in YYYY-MM-DD format (defaults to last day of current month)"
    }
  },
  "additionalProperties": false
}`)

var searchCashflowsSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "enum": ["income", "expense"],
      "description": "Required cashflow type: 'income' or 'expense'"
    },
    "dateFrom": {
      "type": "string",
      "description": "Optional start date in YYYY-MM-DD format"
    },
    "dateTo": {
      "type": "string",
      "description": "Optional end date in YYYY-MM-DD format"
    },
    "status": {
      "type": "string",
      "enum": ["draft", "posted"],
      "description": "Optional document status: 'draft' or 'posted'"
    },
    "query": {
      "type": "string",
      "description": "Optional search term for description or reference"
    },
    "paymentAccount": {
      "type": "string",
      "description": "Optional payment account code or name filter"
    },
    "categoryAccount": {
      "type": "string",
      "description": "Optional category/line account code or name filter"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 50,
      "description": "Optional maximum number of records to return (1-50, default 20)"
    },
    "cursor": {
      "type": "string",
      "description": "Optional pagination cursor"
    }
  },
  "required": ["type"],
  "additionalProperties": false
}`)

var getAccountLedgerSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "account": {
      "type": "string",
      "description": "Required account identifier: account code (e.g. '1101'), name (e.g. 'Kas Kecil'), or numeric ID"
    },
    "page": {
      "type": "integer",
      "minimum": 1,
      "description": "Optional page number (>= 1, default 1)"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 50,
      "description": "Optional items per page (1-50, default 20)"
    }
  },
  "required": ["account"],
  "additionalProperties": false
}`)

var searchTransactionsSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Optional text search across description, reference, or transaction number"
    },
    "reference": {
      "type": "string",
      "description": "Optional exact or partial reference number"
    },
    "source": {
      "type": "string",
      "enum": ["MANUAL", "IMPORT", "SYSTEM"],
      "description": "Optional transaction source filter"
    },
    "dateFrom": {
      "type": "string",
      "description": "Optional start date in YYYY-MM-DD format"
    },
    "dateTo": {
      "type": "string",
      "description": "Optional end date in YYYY-MM-DD format"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 50,
      "description": "Optional maximum number of records to return (1-50, default 20)"
    },
    "cursor": {
      "type": "string",
      "description": "Optional pagination cursor"
    }
  },
  "additionalProperties": false
}`)

var lookupExchangeRateSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "fromCurrency": {
      "type": "string",
      "description": "Required source currency code (e.g. 'USD')"
    },
    "toCurrency": {
      "type": "string",
      "description": "Required target currency code (e.g. 'IDR')"
    },
    "date": {
      "type": "string",
      "description": "Optional effective date in YYYY-MM-DD format (defaults to latest available)"
    }
  },
  "required": ["fromCurrency", "toCurrency"],
  "additionalProperties": false
}`)

// ListAccountsTool returns the llmx.Tool definition for list_accounts.
func ListAccountsTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolListAccounts,
		Description: "List all active postable accounts in the workspace for selection in draft proposals.",
		Schema:      listAccountsSchema,
	}
}

// ListCurrenciesTool returns the llmx.Tool definition for list_currencies.
func ListCurrenciesTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolListCurrencies,
		Description: "List supported currencies in the workspace.",
		Schema:      listCurrenciesSchema,
	}
}

// ListPeriodsTool returns the llmx.Tool definition for list_periods.
func ListPeriodsTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolListPeriods,
		Description: "List open accounting periods in the workspace.",
		Schema:      listPeriodsSchema,
	}
}

// ProposeDraftTool returns the llmx.Tool definition for propose_draft.
func ProposeDraftTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolProposeDraft,
		Description: "Propose a draft financial transaction (expense, income, or journal) for user review without writing to the database.",
		Schema:      proposeDraftSchema,
	}
}

// GetAccountBalancesTool returns the llmx.Tool definition for get_account_balances.
func GetAccountBalancesTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolGetAccountBalances,
		Description: "Query account balances and trial balances as of a specific date with optional type or code filtering. Returns server-aggregated totals and matched net total.",
		Schema:      getAccountBalancesSchema,
	}
}

// GetFinancialSummaryTool returns the llmx.Tool definition for get_financial_summary.
func GetFinancialSummaryTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolGetFinancialSummary,
		Description: "Get high-level financial summary (assets, liabilities, equity, revenue, expense, net income) and monthly revenue/expense trends for a given period.",
		Schema:      getFinancialSummarySchema,
	}
}

// GetProfitLossBreakdownTool returns the llmx.Tool definition for get_profit_loss_breakdown.
func GetProfitLossBreakdownTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolGetProfitLossBreakdown,
		Description: "Get category-level Profit & Loss breakdown (revenues, expenses, net profit) over a date range with exact server-calculated decimal totals.",
		Schema:      getProfitLossBreakdownSchema,
	}
}

// SearchCashflowsTool returns the llmx.Tool definition for search_cashflows.
func SearchCashflowsTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolSearchCashflows,
		Description: "Search income or expense cashflow documents by date range, status, payment account, category account, or text query. Returns matching documents and server-computed total base amount.",
		Schema:      searchCashflowsSchema,
	}
}

// GetAccountLedgerTool returns the llmx.Tool definition for get_account_ledger.
func GetAccountLedgerTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolGetAccountLedger,
		Description: "Get detailed account ledger (mutasi / buku besar) showing historical debit/credit postings and running balances for a specific account.",
		Schema:      getAccountLedgerSchema,
	}
}

// SearchTransactionsTool returns the llmx.Tool definition for search_transactions.
func SearchTransactionsTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolSearchTransactions,
		Description: "Search general journal transactions by query, reference, source, or date range.",
		Schema:      searchTransactionsSchema,
	}
}

// LookupExchangeRateTool returns the llmx.Tool definition for lookup_exchange_rate.
func LookupExchangeRateTool() llmx.Tool {
	return llmx.Tool{
		Name:        ToolLookupExchangeRate,
		Description: "Lookup exchange rate between two currencies for a specific date or retrieve latest rate.",
		Schema:      lookupExchangeRateSchema,
	}
}

// AgentTools returns the standard tools available to the Talk-to-Agent assistant.
func AgentTools() []llmx.Tool {
	return []llmx.Tool{
		ListAccountsTool(),
		ListCurrenciesTool(),
		ListPeriodsTool(),
		ProposeDraftTool(),
		GetAccountBalancesTool(),
		GetFinancialSummaryTool(),
		GetProfitLossBreakdownTool(),
		SearchCashflowsTool(),
		GetAccountLedgerTool(),
		SearchTransactionsTool(),
		LookupExchangeRateTool(),
	}
}

// Tool response DTOs.
type ToolErrorResponse struct {
	Error string `json:"error"`
}

type ListAccountsResponse struct {
	Accounts []AccountOption `json:"accounts"`
}

type ListCurrenciesResponse struct {
	Currencies []CurrencyOption `json:"currencies"`
}

type ListPeriodsResponse struct {
	Periods []PeriodOption `json:"periods"`
}

type ProposeDraftResponse struct {
	Status   string              `json:"status"`
	Proposal model.DraftProposal `json:"proposal"`
}

func formatToolError(msg string) string {
	b, _ := json.Marshal(ToolErrorResponse{Error: msg})
	return string(b)
}

type rawProposalInput struct {
	Kind             string          `json:"kind"`
	TransactionDate  string          `json:"transactionDate"`
	Description      string          `json:"description"`
	Reference        string          `json:"reference"`
	Currency         string          `json:"currency"`
	ExchangeRate     json.RawMessage `json:"exchangeRate"`
	PaymentAccountID json.RawMessage `json:"paymentAccountId"`
	PeriodID         json.RawMessage `json:"periodId"`
	TotalAmount      json.RawMessage `json:"totalAmount"`
	Items            []rawItemInput  `json:"items"`
	Entries          []rawEntryInput `json:"entries"`
}

type rawItemInput struct {
	AccountID   json.RawMessage `json:"accountId"`
	Description string          `json:"description"`
	Quantity    json.RawMessage `json:"quantity"`
	UnitPrice   json.RawMessage `json:"unitPrice"`
	Amount      json.RawMessage `json:"amount"`
}

type rawEntryInput struct {
	AccountID   json.RawMessage `json:"accountId"`
	Direction   string          `json:"direction"`
	Amount      json.RawMessage `json:"amount"`
	Description string          `json:"description"`
}

func parseInt64Helper(raw json.RawMessage, fieldName string) (int64, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return 0, fmt.Sprintf("%s is required", fieldName)
	}
	var num int64
	if err := json.Unmarshal(raw, &num); err == nil {
		return num, ""
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int64(f), ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, ""
		}
	}
	return 0, fmt.Sprintf("invalid %s: expected integer", fieldName)
}

func parseOptionalInt64Helper(raw json.RawMessage, fieldName string) (*int64, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, ""
	}
	n, errStr := parseInt64Helper(raw, fieldName)
	if errStr != "" {
		return nil, errStr
	}
	return &n, ""
}

func parseStringHelper(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return strings.Trim(trimmed, "\"")
}

func parseBoolHelper(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.ToLower(strings.TrimSpace(s))
		return s == "true" || s == "1" || s == "yes"
	}
	return false
}

func parseIntHelper(raw json.RawMessage, fieldName string) (int, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return 0, fmt.Sprintf("%s is required", fieldName)
	}
	var num int
	if err := json.Unmarshal(raw, &num); err == nil {
		return num, ""
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int(f), ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if n, err := strconv.Atoi(s); err == nil {
			return n, ""
		}
	}
	return 0, fmt.Sprintf("invalid %s: expected integer", fieldName)
}

func parseOptionalIntHelper(raw json.RawMessage, fieldName string) (*int, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, ""
	}
	n, errStr := parseIntHelper(raw, fieldName)
	if errStr != "" {
		return nil, errStr
	}
	return &n, ""
}

func parseDecimalHelper(raw json.RawMessage, fieldName string, isMonetary bool, isZeroDec bool) (decimal.Decimal, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return decimal.Zero, fmt.Sprintf("%s is required", fieldName)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return decimal.Zero, fmt.Sprintf("%s cannot be empty", fieldName)
		}
		d, err := parseDecimalAmount(s, isMonetary, isZeroDec)
		if err == nil {
			return d, ""
		}
		d, err = decimal.NewFromString(s)
		if err == nil {
			return d, ""
		}
		return decimal.Zero, fmt.Sprintf("invalid %s: unable to parse decimal %q", fieldName, s)
	}
	var d decimal.Decimal
	if err := json.Unmarshal(raw, &d); err == nil {
		return d, ""
	}
	return decimal.Zero, fmt.Sprintf("invalid %s: expected number or numeric string", fieldName)
}

func parseOptionalDecimalHelper(raw json.RawMessage, fieldName string, isMonetary bool, isZeroDec bool) (*decimal.Decimal, string) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, ""
	}
	d, errStr := parseDecimalHelper(raw, fieldName, isMonetary, isZeroDec)
	if errStr != "" {
		return nil, errStr
	}
	return &d, ""
}

func parseProposalFromJSON(args json.RawMessage) (*model.DraftProposal, string) {
	trimmed := bytes.TrimSpace(args)
	if len(trimmed) == 0 {
		return nil, "arguments are required for propose_draft"
	}
	var raw rawProposalInput
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, fmt.Sprintf("malformed JSON arguments: %v", err)
	}

	kind := model.DraftKind(strings.ToLower(strings.TrimSpace(raw.Kind)))
	switch kind {
	case model.DraftKindExpense, model.DraftKindIncome, model.DraftKindJournal:
	default:
		return nil, fmt.Sprintf("unsupported draft kind %q: must be expense, income, or journal", raw.Kind)
	}

	proposal := &model.DraftProposal{
		Kind:            kind,
		TransactionDate: strings.TrimSpace(raw.TransactionDate),
		Description:     strings.TrimSpace(raw.Description),
		Reference:       strings.TrimSpace(raw.Reference),
		Currency:        strings.ToUpper(strings.TrimSpace(raw.Currency)),
	}

	isZeroDec := isZeroDecimalCurrency(proposal.Currency)

	if len(raw.ExchangeRate) > 0 {
		rate, errStr := parseOptionalDecimalHelper(raw.ExchangeRate, "exchangeRate", false, false)
		if errStr != "" {
			return nil, errStr
		}
		proposal.ExchangeRate = rate
	}

	if len(raw.PaymentAccountID) > 0 {
		pmtID, errStr := parseOptionalInt64Helper(raw.PaymentAccountID, "paymentAccountId")
		if errStr != "" {
			return nil, errStr
		}
		proposal.PaymentAccountID = pmtID
	}

	if len(raw.PeriodID) > 0 {
		pID, errStr := parseOptionalInt64Helper(raw.PeriodID, "periodId")
		if errStr != "" {
			return nil, errStr
		}
		proposal.PeriodID = pID
	}

	if len(raw.TotalAmount) > 0 {
		tot, errStr := parseDecimalHelper(raw.TotalAmount, "totalAmount", true, isZeroDec)
		if errStr != "" {
			return nil, errStr
		}
		proposal.TotalAmount = tot
	}

	if len(raw.Items) > 0 {
		proposal.Items = make([]model.DraftItem, 0, len(raw.Items))
		for i, it := range raw.Items {
			accID, errStr := parseInt64Helper(it.AccountID, fmt.Sprintf("items[%d].accountId", i))
			if errStr != "" {
				return nil, errStr
			}
			qty, errStr := parseDecimalHelper(it.Quantity, fmt.Sprintf("items[%d].quantity", i), false, false)
			if errStr != "" {
				return nil, errStr
			}
			price, errStr := parseDecimalHelper(it.UnitPrice, fmt.Sprintf("items[%d].unitPrice", i), true, isZeroDec)
			if errStr != "" {
				return nil, errStr
			}
			var amt decimal.Decimal
			if len(it.Amount) > 0 && strings.TrimSpace(string(it.Amount)) != "null" {
				parsedAmt, errStr := parseDecimalHelper(it.Amount, fmt.Sprintf("items[%d].amount", i), true, isZeroDec)
				if errStr != "" {
					return nil, errStr
				}
				amt = parsedAmt
			} else {
				amt = qty.Mul(price)
			}
			proposal.Items = append(proposal.Items, model.DraftItem{
				AccountID:   accID,
				Description: strings.TrimSpace(it.Description),
				Quantity:    qty,
				UnitPrice:   price,
				Amount:      amt,
			})
		}
	}

	if len(raw.Entries) > 0 {
		proposal.Entries = make([]model.DraftJournalEntry, 0, len(raw.Entries))
		for i, e := range raw.Entries {
			accID, errStr := parseInt64Helper(e.AccountID, fmt.Sprintf("entries[%d].accountId", i))
			if errStr != "" {
				return nil, errStr
			}
			amt, errStr := parseDecimalHelper(e.Amount, fmt.Sprintf("entries[%d].amount", i), true, isZeroDec)
			if errStr != "" {
				return nil, errStr
			}
			proposal.Entries = append(proposal.Entries, model.DraftJournalEntry{
				AccountID:   accID,
				Direction:   strings.ToUpper(strings.TrimSpace(e.Direction)),
				Amount:      amt,
				Description: strings.TrimSpace(e.Description),
			})
		}
	}

	return proposal, ""
}

// ValidateDraftProposal validates an untrusted draft proposal against workspace finance reference data.
// It returns the normalized proposal if valid, or an error describing the validation failure.
// NO database write is performed.
func ValidateDraftProposal(ctx context.Context, draftCtx FinanceDraftContext, proposal model.DraftProposal) (*model.DraftProposal, error) {
	if draftCtx == nil {
		return nil, errors.New("finance draft context is not configured")
	}

	// 1. Kind check
	switch proposal.Kind {
	case model.DraftKindExpense, model.DraftKindIncome, model.DraftKindJournal:
	default:
		return nil, fmt.Errorf("unsupported draft kind %q: must be expense, income, or journal", proposal.Kind)
	}

	// 2. Date check
	dStr := strings.TrimSpace(proposal.TransactionDate)
	if dStr == "" {
		return nil, errors.New("transactionDate is required")
	}
	t, err := time.Parse("2006-01-02", dStr)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction date %q: must be YYYY-MM-DD", dStr)
	}
	proposal.TransactionDate = t.Format("2006-01-02")

	// 3. Description check
	proposal.Description = strings.TrimSpace(proposal.Description)
	if proposal.Description == "" {
		return nil, errors.New("description is required")
	}

	// 4. Currency check
	proposal.Currency = strings.ToUpper(strings.TrimSpace(proposal.Currency))
	if proposal.Currency == "" {
		return nil, errors.New("currency is required")
	}
	currencies, err := draftCtx.ListCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}
	var matchedCurrency *CurrencyOption
	for _, c := range currencies {
		if strings.EqualFold(c.Code, proposal.Currency) {
			matchedCurrency = &c
			break
		}
	}
	if matchedCurrency == nil {
		return nil, fmt.Errorf("currency %q is not supported in workspace", proposal.Currency)
	}
	proposal.Currency = matchedCurrency.Code

	// 5. Accounts check
	accounts, err := draftCtx.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	accountsByID := make(map[int64]AccountOption, len(accounts))
	accountsByCode := make(map[string]AccountOption, len(accounts))
	for _, acc := range accounts {
		accountsByID[acc.ID] = acc
		if acc.Code != "" {
			accountsByCode[strings.TrimSpace(acc.Code)] = acc
		}
	}

	resolveAccount := func(rawID int64) (int64, bool) {
		if acc, exists := accountsByID[rawID]; exists {
			return acc.ID, true
		}
		codeStr := strconv.FormatInt(rawID, 10)
		if acc, ok := accountsByCode[codeStr]; ok {
			return acc.ID, true
		}
		aliases := map[string][]string{
			"1101": {"1111", "1110"},
			"1111": {"1101"},
			"1102": {"1121", "1120"},
			"1121": {"1102"},
			"5101": {"5100", "6730", "6200"},
			"5100": {"5101", "6730"},
			"6730": {"5101", "5100"},
		}
		if alts, ok := aliases[codeStr]; ok {
			for _, alt := range alts {
				if acc, ok := accountsByCode[alt]; ok {
					return acc.ID, true
				}
			}
		}
		return 0, false
	}
	// 6. Period check (if provided)
	if proposal.PeriodID != nil {
		periods, err := draftCtx.ListPeriods(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list periods: %w", err)
		}
		foundPeriod := false
		for _, p := range periods {
			if p.ID == *proposal.PeriodID {
				foundPeriod = true
				break
			}
		}
		if !foundPeriod {
			return nil, fmt.Errorf("period %d is not an active period in workspace", *proposal.PeriodID)
		}
	}

	// 7. Kind-specific validation
	switch proposal.Kind {
	case model.DraftKindExpense, model.DraftKindIncome:
		if proposal.PaymentAccountID == nil {
			return nil, fmt.Errorf("paymentAccountId is required for %s draft", proposal.Kind)
		}
		if resID, ok := resolveAccount(*proposal.PaymentAccountID); ok {
			proposal.PaymentAccountID = &resID
		} else {
			return nil, fmt.Errorf("account %d does not exist in workspace", *proposal.PaymentAccountID)
		}

		if len(proposal.Items) == 0 {
			return nil, fmt.Errorf("at least 1 item is required for %s draft", proposal.Kind)
		}

		sumItems := decimal.Zero
		for i := range proposal.Items {
			it := &proposal.Items[i]
			if resID, ok := resolveAccount(it.AccountID); ok {
				it.AccountID = resID
			} else {
				return nil, fmt.Errorf("account %d does not exist in workspace", it.AccountID)
			}
			if it.Quantity.LessThanOrEqual(decimal.Zero) {
				return nil, fmt.Errorf("item %d quantity must be greater than zero", i+1)
			}
			if it.UnitPrice.LessThan(decimal.Zero) {
				return nil, fmt.Errorf("item %d unit price cannot be negative", i+1)
			}
			expectedAmount := it.Quantity.Mul(it.UnitPrice)
			if it.Amount.IsZero() {
				it.Amount = expectedAmount
			}
			sumItems = sumItems.Add(it.Amount)
		}

		if proposal.TotalAmount.IsZero() {
			proposal.TotalAmount = sumItems
		} else if !proposal.TotalAmount.Equal(sumItems) {
			diff := proposal.TotalAmount.Sub(sumItems).Abs()
			if diff.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
				proposal.TotalAmount = sumItems
			} else {
				return nil, fmt.Errorf("item total mismatch: sum of items (%s) does not match total amount (%s)", sumItems.String(), proposal.TotalAmount.String())
			}
		}

		if proposal.TotalAmount.LessThanOrEqual(decimal.Zero) {
			return nil, errors.New("total amount must be greater than zero")
		}

		proposal.Entries = nil

	case model.DraftKindJournal:
		if len(proposal.Entries) < 2 {
			return nil, errors.New("journal draft requires at least 2 entries")
		}

		totalDebit := decimal.Zero
		totalCredit := decimal.Zero
		debitCount := 0
		creditCount := 0

		for i := range proposal.Entries {
			e := &proposal.Entries[i]
			if resID, ok := resolveAccount(e.AccountID); ok {
				e.AccountID = resID
			} else {
				return nil, fmt.Errorf("account %d does not exist in workspace", e.AccountID)
			}
			dir := strings.ToUpper(strings.TrimSpace(e.Direction))
			if dir != "DEBIT" && dir != "CREDIT" {
				return nil, fmt.Errorf("entry %d invalid direction %q: must be DEBIT or CREDIT", i+1, e.Direction)
			}
			e.Direction = dir
			if e.Amount.LessThanOrEqual(decimal.Zero) {
				return nil, fmt.Errorf("entry %d amount must be greater than zero", i+1)
			}
			if dir == "DEBIT" {
				debitCount++
				totalDebit = totalDebit.Add(e.Amount)
			} else {
				creditCount++
				totalCredit = totalCredit.Add(e.Amount)
			}
		}

		if debitCount == 0 || creditCount == 0 {
			return nil, errors.New("journal draft requires at least one DEBIT and at least one CREDIT entry")
		}
		if !totalDebit.Equal(totalCredit) {
			return nil, fmt.Errorf("unbalanced journal draft: total debits (%s) does not equal total credits (%s)", totalDebit.String(), totalCredit.String())
		}

		if proposal.TotalAmount.IsZero() {
			proposal.TotalAmount = totalDebit
		} else if !proposal.TotalAmount.Equal(totalDebit) {
			diff := proposal.TotalAmount.Sub(totalDebit).Abs()
			if diff.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
				proposal.TotalAmount = totalDebit
			} else {
				return nil, fmt.Errorf("total amount (%s) does not match balanced journal total (%s)", proposal.TotalAmount.String(), totalDebit.String())
			}
		}

		if proposal.TotalAmount.LessThanOrEqual(decimal.Zero) {
			return nil, errors.New("total amount must be greater than zero")
		}

		proposal.Items = nil
	}

	return &proposal, nil
}

// ExecuteTool executes an agent tool call against the given FinanceDraftContext.
// It returns:
// - result: JSON formatted string response to return to the model (or structured error JSON on argument/validation failures).
// - proposal: validated DraftProposal if propose_draft succeeded, nil otherwise.
// - err: internal execution error if any (e.g. context cancellation or storage failure). Model-level errors are returned as JSON in result with err == nil.
func ExecuteTool(ctx context.Context, draftCtx FinanceDraftContext, call llmx.ToolCall) (string, *model.DraftProposal, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if draftCtx == nil {
		return "", nil, errors.New("finance draft context is not configured")
	}

	switch call.Name {
	case ToolListAccounts:
		accs, err := draftCtx.ListAccounts(ctx)
		if err != nil {
			return "", nil, err
		}
		b, err := json.Marshal(ListAccountsResponse{Accounts: accs})
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolListCurrencies:
		currs, err := draftCtx.ListCurrencies(ctx)
		if err != nil {
			return "", nil, err
		}
		b, err := json.Marshal(ListCurrenciesResponse{Currencies: currs})
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolListPeriods:
		periods, err := draftCtx.ListPeriods(ctx)
		if err != nil {
			return "", nil, err
		}
		b, err := json.Marshal(ListPeriodsResponse{Periods: periods})
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolProposeDraft:
		rawProposal, errStr := parseProposalFromJSON(call.Arguments)
		if errStr != "" {
			return formatToolError(errStr), nil, nil
		}
		validated, err := ValidateDraftProposal(ctx, draftCtx, *rawProposal)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		resp := ProposeDraftResponse{
			Status:   "proposed",
			Proposal: *validated,
		}
		b, err := json.Marshal(resp)
		if err != nil {
			return "", nil, err
		}
		return string(b), validated, nil

	case ToolGetAccountBalances:
		var raw struct {
			AsOfDate                    json.RawMessage `json:"asOfDate"`
			Account                     json.RawMessage `json:"account"`
			Type                        json.RawMessage `json:"type"`
			IncludePerCurrencyBreakdown json.RawMessage `json:"includePerCurrencyBreakdown"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		query := AccountBalanceQuery{
			AsOfDate:                    parseStringHelper(raw.AsOfDate),
			Account:                     parseStringHelper(raw.Account),
			Type:                        strings.ToUpper(parseStringHelper(raw.Type)),
			IncludePerCurrencyBreakdown: parseBoolHelper(raw.IncludePerCurrencyBreakdown),
		}
		res, err := draftCtx.GetAccountBalances(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolGetFinancialSummary:
		var raw struct {
			FromMonth       json.RawMessage `json:"fromMonth"`
			ToMonth         json.RawMessage `json:"toMonth"`
			ComparePrevious json.RawMessage `json:"comparePrevious"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		query := FinancialSummaryQuery{
			FromMonth:       parseStringHelper(raw.FromMonth),
			ToMonth:         parseStringHelper(raw.ToMonth),
			ComparePrevious: parseBoolHelper(raw.ComparePrevious),
		}
		res, err := draftCtx.GetFinancialSummary(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolGetProfitLossBreakdown:
		var raw struct {
			DateFrom json.RawMessage `json:"dateFrom"`
			DateTo   json.RawMessage `json:"dateTo"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		query := ProfitLossQuery{
			DateFrom: parseStringHelper(raw.DateFrom),
			DateTo:   parseStringHelper(raw.DateTo),
		}
		res, err := draftCtx.GetProfitLossBreakdown(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolSearchCashflows:
		var raw struct {
			Type            json.RawMessage `json:"type"`
			DateFrom        json.RawMessage `json:"dateFrom"`
			DateTo          json.RawMessage `json:"dateTo"`
			Status          json.RawMessage `json:"status"`
			Query           json.RawMessage `json:"query"`
			PaymentAccount  json.RawMessage `json:"paymentAccount"`
			CategoryAccount json.RawMessage `json:"categoryAccount"`
			Limit           json.RawMessage `json:"limit"`
			Cursor          json.RawMessage `json:"cursor"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		cfType := strings.ToLower(parseStringHelper(raw.Type))
		if cfType == "" {
			return formatToolError("type is required (must be 'income' or 'expense')"), nil, nil
		}
		if cfType != "income" && cfType != "expense" {
			return formatToolError(fmt.Sprintf("invalid type %q: must be 'income' or 'expense'", cfType)), nil, nil
		}
		var limit int
		if len(raw.Limit) > 0 {
			lim, errStr := parseOptionalIntHelper(raw.Limit, "limit")
			if errStr != "" {
				return formatToolError(errStr), nil, nil
			}
			if lim != nil {
				limit = *lim
			}
		}
		query := CashflowSearchQuery{
			Type:            cfType,
			DateFrom:        parseStringHelper(raw.DateFrom),
			DateTo:          parseStringHelper(raw.DateTo),
			Status:          strings.ToLower(parseStringHelper(raw.Status)),
			Query:           parseStringHelper(raw.Query),
			PaymentAccount:  parseStringHelper(raw.PaymentAccount),
			CategoryAccount: parseStringHelper(raw.CategoryAccount),
			Limit:           limit,
			Cursor:          parseStringHelper(raw.Cursor),
		}
		res, err := draftCtx.SearchCashflows(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolGetAccountLedger:
		var raw struct {
			Account json.RawMessage `json:"account"`
			Page    json.RawMessage `json:"page"`
			Limit   json.RawMessage `json:"limit"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		account := parseStringHelper(raw.Account)
		if account == "" {
			return formatToolError("account is required"), nil, nil
		}
		var page int
		if len(raw.Page) > 0 {
			p, errStr := parseOptionalIntHelper(raw.Page, "page")
			if errStr != "" {
				return formatToolError(errStr), nil, nil
			}
			if p != nil {
				page = *p
			}
		}
		var limit int
		if len(raw.Limit) > 0 {
			lim, errStr := parseOptionalIntHelper(raw.Limit, "limit")
			if errStr != "" {
				return formatToolError(errStr), nil, nil
			}
			if lim != nil {
				limit = *lim
			}
		}
		query := AccountLedgerQuery{
			Account: account,
			Page:    page,
			Limit:   limit,
		}
		res, err := draftCtx.GetAccountLedger(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolSearchTransactions:
		var raw struct {
			Query     json.RawMessage `json:"query"`
			Reference json.RawMessage `json:"reference"`
			Source    json.RawMessage `json:"source"`
			DateFrom  json.RawMessage `json:"dateFrom"`
			DateTo    json.RawMessage `json:"dateTo"`
			Limit     json.RawMessage `json:"limit"`
			Cursor    json.RawMessage `json:"cursor"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		var limit int
		if len(raw.Limit) > 0 {
			lim, errStr := parseOptionalIntHelper(raw.Limit, "limit")
			if errStr != "" {
				return formatToolError(errStr), nil, nil
			}
			if lim != nil {
				limit = *lim
			}
		}
		query := TransactionSearchQuery{
			Query:     parseStringHelper(raw.Query),
			Reference: parseStringHelper(raw.Reference),
			Source:    strings.ToUpper(parseStringHelper(raw.Source)),
			DateFrom:  parseStringHelper(raw.DateFrom),
			DateTo:    parseStringHelper(raw.DateTo),
			Limit:     limit,
			Cursor:    parseStringHelper(raw.Cursor),
		}
		res, err := draftCtx.SearchTransactions(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	case ToolLookupExchangeRate:
		var raw struct {
			FromCurrency json.RawMessage `json:"fromCurrency"`
			ToCurrency   json.RawMessage `json:"toCurrency"`
			Date         json.RawMessage `json:"date"`
		}
		if len(bytes.TrimSpace(call.Arguments)) > 0 {
			if err := json.Unmarshal(call.Arguments, &raw); err != nil {
				return formatToolError(fmt.Sprintf("invalid arguments: %v", err)), nil, nil
			}
		}
		fromCurr := strings.ToUpper(parseStringHelper(raw.FromCurrency))
		toCurr := strings.ToUpper(parseStringHelper(raw.ToCurrency))
		if fromCurr == "" {
			return formatToolError("fromCurrency is required"), nil, nil
		}
		if toCurr == "" {
			return formatToolError("toCurrency is required"), nil, nil
		}
		query := ExchangeRateQuery{
			FromCurrency: fromCurr,
			ToCurrency:   toCurr,
			Date:         parseStringHelper(raw.Date),
		}
		res, err := draftCtx.LookupExchangeRate(ctx, query)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", nil, err
			}
			return formatToolError(err.Error()), nil, nil
		}
		b, err := json.Marshal(res)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil

	default:
		return formatToolError(fmt.Sprintf("unknown tool: %q", call.Name)), nil, nil
	}
}
