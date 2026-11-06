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
	ToolListAccounts   = "list_accounts"
	ToolListCurrencies = "list_currencies"
	ToolListPeriods    = "list_periods"
	ToolProposeDraft   = "propose_draft"
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

// AgentTools returns the standard tools available to the Talk-to-Agent assistant.
func AgentTools() []llmx.Tool {
	return []llmx.Tool{
		ListAccountsTool(),
		ListCurrenciesTool(),
		ListPeriodsTool(),
		ProposeDraftTool(),
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
	for _, acc := range accounts {
		accountsByID[acc.ID] = acc
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
		if _, exists := accountsByID[*proposal.PaymentAccountID]; !exists {
			return nil, fmt.Errorf("account %d does not exist in workspace", *proposal.PaymentAccountID)
		}

		if len(proposal.Items) == 0 {
			return nil, fmt.Errorf("at least 1 item is required for %s draft", proposal.Kind)
		}

		sumItems := decimal.Zero
		for i := range proposal.Items {
			it := &proposal.Items[i]
			if _, exists := accountsByID[it.AccountID]; !exists {
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
			if _, exists := accountsByID[e.AccountID]; !exists {
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

	default:
		return formatToolError(fmt.Sprintf("unknown tool: %q", call.Name)), nil, nil
	}
}
