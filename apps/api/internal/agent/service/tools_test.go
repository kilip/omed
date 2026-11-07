package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	agentsvc "github.com/kilip/omed/api/internal/agent/service"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFinanceDraftContext struct {
	accounts      []agentsvc.AccountOption
	accountsErr   error
	currencies    []agentsvc.CurrencyOption
	currenciesErr error
	periods       []agentsvc.PeriodOption
	periodsErr    error

	saveDraftCalls int
	saveDraftErr   error
	saveDraftID    int64

	accountBalancesRes  agentsvc.AccountBalanceResult
	accountBalancesErr  error
	financialSummaryRes agentsvc.FinancialSummaryResult
	financialSummaryErr error
	profitLossRes       agentsvc.ProfitLossResult
	profitLossErr       error
	cashflowsRes        agentsvc.CashflowSearchResult
	cashflowsErr        error
	ledgerRes           agentsvc.AccountLedgerResult
	ledgerErr           error
	transactionsRes     agentsvc.TransactionSearchResult
	transactionsErr     error
	exchangeRateRes     agentsvc.ExchangeRateResult
	exchangeRateErr     error
}

func (m *mockFinanceDraftContext) ListAccounts(ctx context.Context) ([]agentsvc.AccountOption, error) {
	if m.accountsErr != nil {
		return nil, m.accountsErr
	}
	return m.accounts, nil
}

func (m *mockFinanceDraftContext) ListCurrencies(ctx context.Context) ([]agentsvc.CurrencyOption, error) {
	if m.currenciesErr != nil {
		return nil, m.currenciesErr
	}
	return m.currencies, nil
}

func (m *mockFinanceDraftContext) ListPeriods(ctx context.Context) ([]agentsvc.PeriodOption, error) {
	if m.periodsErr != nil {
		return nil, m.periodsErr
	}
	return m.periods, nil
}

func (m *mockFinanceDraftContext) SaveDraft(ctx context.Context, proposal model.DraftProposal) (int64, error) {
	m.saveDraftCalls++
	if m.saveDraftErr != nil {
		return 0, m.saveDraftErr
	}
	return m.saveDraftID, nil
}

func (m *mockFinanceDraftContext) GetAccountBalances(ctx context.Context, req agentsvc.AccountBalanceQuery) (agentsvc.AccountBalanceResult, error) {
	if m.accountBalancesErr != nil {
		return agentsvc.AccountBalanceResult{}, m.accountBalancesErr
	}
	if len(m.accountBalancesRes.Accounts) > 0 || m.accountBalancesRes.BaseCurrency != "" {
		return m.accountBalancesRes, nil
	}
	return agentsvc.AccountBalanceResult{
		BaseCurrency: "IDR",
		Accounts:     []agentsvc.AccountBalanceItem{},
	}, nil
}

func (m *mockFinanceDraftContext) GetFinancialSummary(ctx context.Context, req agentsvc.FinancialSummaryQuery) (agentsvc.FinancialSummaryResult, error) {
	if m.financialSummaryErr != nil {
		return agentsvc.FinancialSummaryResult{}, m.financialSummaryErr
	}
	if m.financialSummaryRes.BaseCurrency != "" {
		return m.financialSummaryRes, nil
	}
	return agentsvc.FinancialSummaryResult{
		BaseCurrency: "IDR",
	}, nil
}

func (m *mockFinanceDraftContext) GetProfitLossBreakdown(ctx context.Context, req agentsvc.ProfitLossQuery) (agentsvc.ProfitLossResult, error) {
	if m.profitLossErr != nil {
		return agentsvc.ProfitLossResult{}, m.profitLossErr
	}
	if m.profitLossRes.BaseCurrency != "" {
		return m.profitLossRes, nil
	}
	return agentsvc.ProfitLossResult{
		BaseCurrency: "IDR",
		Revenues:     []agentsvc.ProfitLossCategoryItem{},
		Expenses:     []agentsvc.ProfitLossCategoryItem{},
	}, nil
}

func (m *mockFinanceDraftContext) SearchCashflows(ctx context.Context, req agentsvc.CashflowSearchQuery) (agentsvc.CashflowSearchResult, error) {
	if m.cashflowsErr != nil {
		return agentsvc.CashflowSearchResult{}, m.cashflowsErr
	}
	if len(m.cashflowsRes.Items) > 0 || !m.cashflowsRes.ReturnedBaseAmountSum.IsZero() {
		return m.cashflowsRes, nil
	}
	return agentsvc.CashflowSearchResult{
		Items: []agentsvc.CashflowDocItem{},
	}, nil
}

func (m *mockFinanceDraftContext) GetAccountLedger(ctx context.Context, req agentsvc.AccountLedgerQuery) (agentsvc.AccountLedgerResult, error) {
	if m.ledgerErr != nil {
		return agentsvc.AccountLedgerResult{}, m.ledgerErr
	}
	if m.ledgerRes.AccountCode != "" || len(m.ledgerRes.AmbiguousMatches) > 0 || len(m.ledgerRes.Postings) > 0 {
		return m.ledgerRes, nil
	}
	return agentsvc.AccountLedgerResult{
		AccountCode:  req.Account,
		BaseCurrency: "IDR",
		Postings:     []agentsvc.AccountLedgerItem{},
	}, nil
}

func (m *mockFinanceDraftContext) SearchTransactions(ctx context.Context, req agentsvc.TransactionSearchQuery) (agentsvc.TransactionSearchResult, error) {
	if m.transactionsErr != nil {
		return agentsvc.TransactionSearchResult{}, m.transactionsErr
	}
	if len(m.transactionsRes.Items) > 0 {
		return m.transactionsRes, nil
	}
	return agentsvc.TransactionSearchResult{
		Items: []agentsvc.TransactionItem{},
	}, nil
}

func (m *mockFinanceDraftContext) LookupExchangeRate(ctx context.Context, req agentsvc.ExchangeRateQuery) (agentsvc.ExchangeRateResult, error) {
	if m.exchangeRateErr != nil {
		return agentsvc.ExchangeRateResult{}, m.exchangeRateErr
	}
	if !m.exchangeRateRes.Rate.IsZero() || len(m.exchangeRateRes.Rates) > 0 {
		return m.exchangeRateRes, nil
	}
	return agentsvc.ExchangeRateResult{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
	}, nil
}

func newTestMockContext() *mockFinanceDraftContext {
	return &mockFinanceDraftContext{
		accounts: []agentsvc.AccountOption{
			{ID: 101, Code: "1101", Name: "Kas Utama"},
			{ID: 102, Code: "1102", Name: "Bank Kaltimtara"},
			{ID: 201, Code: "5101", Name: "Beban Operasional & Perlengkapan"},
			{ID: 202, Code: "5102", Name: "Beban Konsumsi"},
			{ID: 301, Code: "4101", Name: "Pendapatan Jasa"},
		},
		currencies: []agentsvc.CurrencyOption{
			{Code: "IDR", Name: "Indonesian Rupiah", Symbol: "Rp", Decimals: 0},
			{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2},
		},
		periods: []agentsvc.PeriodOption{
			{
				ID:         1,
				Name:       "January 2026",
				StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:    time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
				FiscalYear: 2026,
				Status:     "open",
			},
			{
				ID:         2,
				Name:       "February 2026",
				StartDate:  time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				EndDate:    time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
				FiscalYear: 2026,
				Status:     "open",
			},
		},
	}
}

func TestAgentTools_Definitions(t *testing.T) {
	tools := agentsvc.AgentTools()
	require.Len(t, tools, 11)

	toolNames := make(map[string]llmx.Tool)
	for _, tool := range tools {
		toolNames[tool.Name] = tool
	}

	expectedNames := []string{
		agentsvc.ToolListAccounts,
		agentsvc.ToolListCurrencies,
		agentsvc.ToolListPeriods,
		agentsvc.ToolProposeDraft,
		agentsvc.ToolGetAccountBalances,
		agentsvc.ToolGetFinancialSummary,
		agentsvc.ToolGetProfitLossBreakdown,
		agentsvc.ToolSearchCashflows,
		agentsvc.ToolGetAccountLedger,
		agentsvc.ToolSearchTransactions,
		agentsvc.ToolLookupExchangeRate,
	}
	for _, name := range expectedNames {
		tool, exists := toolNames[name]
		assert.True(t, exists, "tool %s must exist", name)
		assert.NotEmpty(t, tool.Description, "tool %s must have description", name)
		assert.NotEmpty(t, tool.Schema, "tool %s must have schema", name)

		var schemaObj map[string]any
		err := json.Unmarshal(tool.Schema, &schemaObj)
		require.NoError(t, err, "tool %s schema must be valid JSON", name)
		assert.Equal(t, "object", schemaObj["type"])
	}

	// Verify propose_draft schema details
	proposeTool := toolNames[agentsvc.ToolProposeDraft]
	var proposeSchema map[string]any
	err := json.Unmarshal(proposeTool.Schema, &proposeSchema)
	require.NoError(t, err)

	props, ok := proposeSchema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "kind")
	assert.Contains(t, props, "transactionDate")
	assert.Contains(t, props, "description")
	assert.Contains(t, props, "currency")
	assert.Contains(t, props, "totalAmount")
	assert.Contains(t, props, "items")
	assert.Contains(t, props, "entries")

	reqList, ok := proposeSchema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, reqList, "kind")
	assert.Contains(t, reqList, "transactionDate")
	assert.Contains(t, reqList, "currency")
	assert.Contains(t, reqList, "totalAmount")
}

func TestExecuteTool_ListAccounts(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_1",
		Name:      agentsvc.ToolListAccounts,
		Arguments: json.RawMessage(`{}`),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls, "SaveDraft must not be called")

	var accountsResp agentsvc.ListAccountsResponse
	err = json.Unmarshal([]byte(res), &accountsResp)
	require.NoError(t, err)
	assert.Len(t, accountsResp.Accounts, 5)
	assert.Equal(t, int64(101), accountsResp.Accounts[0].ID)
	assert.Equal(t, "1101", accountsResp.Accounts[0].Code)
	assert.Equal(t, "Kas Utama", accountsResp.Accounts[0].Name)
}

func TestExecuteTool_ListCurrencies(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_2",
		Name:      agentsvc.ToolListCurrencies,
		Arguments: json.RawMessage(`{}`),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls, "SaveDraft must not be called")

	var currResp agentsvc.ListCurrenciesResponse
	err = json.Unmarshal([]byte(res), &currResp)
	require.NoError(t, err)
	assert.Len(t, currResp.Currencies, 2)
	assert.Equal(t, "IDR", currResp.Currencies[0].Code)
	assert.Equal(t, "USD", currResp.Currencies[1].Code)
}

func TestExecuteTool_ListPeriods(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_3",
		Name:      agentsvc.ToolListPeriods,
		Arguments: json.RawMessage(`{}`),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls, "SaveDraft must not be called")

	var periodsResp agentsvc.ListPeriodsResponse
	err = json.Unmarshal([]byte(res), &periodsResp)
	require.NoError(t, err)
	assert.Len(t, periodsResp.Periods, 2)
	assert.Equal(t, int64(1), periodsResp.Periods[0].ID)
	assert.Equal(t, "January 2026", periodsResp.Periods[0].Name)
}

// Reference Case 1: Bank Kaltimtara ATM slip (Rp 2.000.000, journal draft: Dr Cash on Hand / Cr Bank account).
func TestExecuteTool_ReferenceCase1_BankKaltimtaraATMSlip(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "journal",
		"transactionDate": "2026-01-15",
		"description": "Tarik Tunai ATM Bank Kaltimtara",
		"reference": "ATM-KALTIMTARA-001",
		"currency": "IDR",
		"periodId": 1,
		"totalAmount": "2000000.00",
		"entries": [
			{
				"accountId": 101,
				"direction": "DEBIT",
				"amount": "2000000.00",
				"description": "Kas di tangan (Cash on Hand)"
			},
			{
				"accountId": 102,
				"direction": "CREDIT",
				"amount": "2000000.00",
				"description": "Rekening Bank Kaltimtara"
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_ref1",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	require.NotNil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls, "ZERO-WRITE GUARANTEE: SaveDraft must NOT be invoked")

	assert.Equal(t, model.DraftKindJournal, proposal.Kind)
	assert.Equal(t, "2026-01-15", proposal.TransactionDate)
	assert.Equal(t, "Tarik Tunai ATM Bank Kaltimtara", proposal.Description)
	assert.Equal(t, "ATM-KALTIMTARA-001", proposal.Reference)
	assert.Equal(t, "IDR", proposal.Currency)
	require.NotNil(t, proposal.PeriodID)
	assert.Equal(t, int64(1), *proposal.PeriodID)
	assert.True(t, proposal.TotalAmount.Equal(decimal.NewFromInt(2000000)))

	require.Len(t, proposal.Entries, 2)
	assert.Equal(t, int64(101), proposal.Entries[0].AccountID)
	assert.Equal(t, "DEBIT", proposal.Entries[0].Direction)
	assert.True(t, proposal.Entries[0].Amount.Equal(decimal.NewFromInt(2000000)))

	assert.Equal(t, int64(102), proposal.Entries[1].AccountID)
	assert.Equal(t, "CREDIT", proposal.Entries[1].Direction)
	assert.True(t, proposal.Entries[1].Amount.Equal(decimal.NewFromInt(2000000)))

	// Check response JSON returned to model
	var resp agentsvc.ProposeDraftResponse
	err = json.Unmarshal([]byte(res), &resp)
	require.NoError(t, err)
	assert.Equal(t, "proposed", resp.Status)
	assert.Equal(t, model.DraftKindJournal, resp.Proposal.Kind)
	assert.True(t, resp.Proposal.TotalAmount.Equal(decimal.NewFromInt(2000000)))
}

// Reference Case 1 variant: Currency format tolerance (e.g. "Rp 2.000.000" and integer IDs).
func TestExecuteTool_ReferenceCase1_FormattedAmountsAndNumbers(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "journal",
		"transactionDate": "2026-01-15",
		"description": "Tarik Tunai ATM",
		"currency": "IDR",
		"totalAmount": "Rp 2.000.000",
		"entries": [
			{
				"accountId": 101,
				"direction": "DEBIT",
				"amount": 2000000
			},
			{
				"accountId": 102,
				"direction": "CREDIT",
				"amount": "2.000.000"
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_ref1_formatted",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	require.NotNil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)
	assert.True(t, proposal.TotalAmount.Equal(decimal.NewFromInt(2000000)))
	assert.Contains(t, res, `"status":"proposed"`)
}

// Reference Case 2: 3-line shop receipt (total Rp 70.000, cash, expense draft with multiple items).
func TestExecuteTool_ReferenceCase2_ThreeLineShopReceipt(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "expense",
		"transactionDate": "2026-01-20",
		"description": "Pembelian perlengkapan toko dan konsumsi",
		"reference": "STR-0701",
		"currency": "IDR",
		"paymentAccountId": 101,
		"totalAmount": "70000.00",
		"items": [
			{
				"accountId": 201,
				"description": "Buku Folio",
				"quantity": "1",
				"unitPrice": "25000",
				"amount": "25000"
			},
			{
				"accountId": 201,
				"description": "Pulpen Gel",
				"quantity": "3",
				"unitPrice": "10000",
				"amount": "30000"
			},
			{
				"accountId": 202,
				"description": "Air Mineral Botol",
				"quantity": "3",
				"unitPrice": "5000",
				"amount": "15000"
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_ref2",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	require.NotNil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls, "ZERO-WRITE GUARANTEE: SaveDraft must NOT be invoked")

	assert.Equal(t, model.DraftKindExpense, proposal.Kind)
	assert.Equal(t, "2026-01-20", proposal.TransactionDate)
	assert.Equal(t, "Pembelian perlengkapan toko dan konsumsi", proposal.Description)
	assert.Equal(t, "STR-0701", proposal.Reference)
	assert.Equal(t, "IDR", proposal.Currency)
	require.NotNil(t, proposal.PaymentAccountID)
	assert.Equal(t, int64(101), *proposal.PaymentAccountID)
	assert.True(t, proposal.TotalAmount.Equal(decimal.NewFromInt(70000)))

	require.Len(t, proposal.Items, 3)
	// Item 1
	assert.Equal(t, int64(201), proposal.Items[0].AccountID)
	assert.Equal(t, "Buku Folio", proposal.Items[0].Description)
	assert.True(t, proposal.Items[0].Quantity.Equal(decimal.NewFromInt(1)))
	assert.True(t, proposal.Items[0].UnitPrice.Equal(decimal.NewFromInt(25000)))
	assert.True(t, proposal.Items[0].Amount.Equal(decimal.NewFromInt(25000)))
	// Item 2
	assert.Equal(t, int64(201), proposal.Items[1].AccountID)
	assert.Equal(t, "Pulpen Gel", proposal.Items[1].Description)
	assert.True(t, proposal.Items[1].Quantity.Equal(decimal.NewFromInt(3)))
	assert.True(t, proposal.Items[1].UnitPrice.Equal(decimal.NewFromInt(10000)))
	assert.True(t, proposal.Items[1].Amount.Equal(decimal.NewFromInt(30000)))
	// Item 3
	assert.Equal(t, int64(202), proposal.Items[2].AccountID)
	assert.Equal(t, "Air Mineral Botol", proposal.Items[2].Description)
	assert.True(t, proposal.Items[2].Quantity.Equal(decimal.NewFromInt(3)))
	assert.True(t, proposal.Items[2].UnitPrice.Equal(decimal.NewFromInt(5000)))
	assert.True(t, proposal.Items[2].Amount.Equal(decimal.NewFromInt(15000)))

	// Check response JSON returned to model
	var resp agentsvc.ProposeDraftResponse
	err = json.Unmarshal([]byte(res), &resp)
	require.NoError(t, err)
	assert.Equal(t, "proposed", resp.Status)
	assert.Equal(t, model.DraftKindExpense, resp.Proposal.Kind)
	assert.True(t, resp.Proposal.TotalAmount.Equal(decimal.NewFromInt(70000)))
}

func TestExecuteTool_IncomeDraftProposal(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "income",
		"transactionDate": "2026-01-25",
		"description": "Konsultasi IT",
		"reference": "INV-INC-001",
		"currency": "IDR",
		"paymentAccountId": 102,
		"totalAmount": "5000000.00",
		"items": [
			{
				"accountId": 301,
				"description": "Jasa Implementasi Sistem",
				"quantity": 1,
				"unitPrice": 5000000
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_income",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	require.NotNil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)
	assert.Equal(t, model.DraftKindIncome, proposal.Kind)
	assert.True(t, proposal.TotalAmount.Equal(decimal.NewFromInt(5000000)))
	assert.Contains(t, res, `"status":"proposed"`)
}

// Error case: Unbalanced journal draft rejection.
func TestExecuteTool_Error_UnbalancedJournalDraft(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "journal",
		"transactionDate": "2026-01-15",
		"description": "Unbalanced transfer",
		"currency": "IDR",
		"totalAmount": "2000000.00",
		"entries": [
			{
				"accountId": 101,
				"direction": "DEBIT",
				"amount": "2000000.00"
			},
			{
				"accountId": 102,
				"direction": "CREDIT",
				"amount": "1500000.00"
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_unbalanced",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err) // Returned as model error in string, not internal Go error
	assert.Nil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)

	var errResp agentsvc.ToolErrorResponse
	err = json.Unmarshal([]byte(res), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "unbalanced journal draft")
	assert.Contains(t, errResp.Error, "total debits (2000000) does not equal total credits (1500000)")
}

// Error case: Journal with only DEBIT entries (no CREDIT).
func TestExecuteTool_Error_JournalSingleDirection(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "journal",
		"transactionDate": "2026-01-15",
		"description": "All debits",
		"currency": "IDR",
		"totalAmount": "2000000.00",
		"entries": [
			{
				"accountId": 101,
				"direction": "DEBIT",
				"amount": "1000000.00"
			},
			{
				"accountId": 102,
				"direction": "DEBIT",
				"amount": "1000000.00"
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_no_credit",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)

	var errResp agentsvc.ToolErrorResponse
	err = json.Unmarshal([]byte(res), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "at least one DEBIT and at least one CREDIT entry")
}

// Error case: Item total mismatch rejection.
func TestExecuteTool_Error_ItemTotalMismatch(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	// Items sum to 70.000, but totalAmount is 80.000
	args := `{
		"kind": "expense",
		"transactionDate": "2026-01-20",
		"description": "Receipt with bad total",
		"currency": "IDR",
		"paymentAccountId": 101,
		"totalAmount": "80000.00",
		"items": [
			{
				"accountId": 201,
				"description": "Item 1",
				"quantity": "1",
				"unitPrice": "40000",
				"amount": "40000"
			},
			{
				"accountId": 202,
				"description": "Item 2",
				"quantity": "1",
				"unitPrice": "30000",
				"amount": "30000"
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_mismatch",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)

	var errResp agentsvc.ToolErrorResponse
	err = json.Unmarshal([]byte(res), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "item total mismatch")
	assert.Contains(t, errResp.Error, "sum of items (70000) does not match total amount (80000)")
}

// Error case: Unknown account ID rejection.
func TestExecuteTool_Error_UnknownAccountID(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	// Case A: Payment account unknown (999)
	t.Run("UnknownPaymentAccount", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Test unknown payment account",
			"currency": "IDR",
			"paymentAccountId": 999,
			"totalAmount": "50000.00",
			"items": [
				{
					"accountId": 201,
					"quantity": 1,
					"unitPrice": 50000
				}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_unknown_pmt",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, `"account 999 does not exist in workspace"`)
	})

	// Case B: Item account unknown (99999)
	t.Run("UnknownItemAccount", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Test unknown item account",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": [
				{
					"accountId": 99999,
					"quantity": 1,
					"unitPrice": 50000
				}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_unknown_item",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, `"account 99999 does not exist in workspace"`)
	})

	// Case C: Journal entry account unknown (8888)
	t.Run("UnknownJournalAccount", func(t *testing.T) {
		args := `{
			"kind": "journal",
			"transactionDate": "2026-01-15",
			"description": "Test unknown journal account",
			"currency": "IDR",
			"totalAmount": "100000.00",
			"entries": [
				{
					"accountId": 101,
					"direction": "DEBIT",
					"amount": 100000
				},
				{
					"accountId": 8888,
					"direction": "CREDIT",
					"amount": 100000
				}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_unknown_journal",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, `"account 8888 does not exist in workspace"`)
	})

	assert.Equal(t, 0, mockCtx.saveDraftCalls)
}

// Error case: Unknown currency.
func TestExecuteTool_Error_UnknownCurrency(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "expense",
		"transactionDate": "2026-01-20",
		"description": "Test unknown currency",
		"currency": "EUR",
		"paymentAccountId": 101,
		"totalAmount": "50.00",
		"items": [
			{
				"accountId": 201,
				"quantity": 1,
				"unitPrice": 50
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_unknown_currency",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	var errResp agentsvc.ToolErrorResponse
	err = json.Unmarshal([]byte(res), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, `currency "EUR" is not supported in workspace`)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)
}

// Error case: Period ID not in open periods list.
func TestExecuteTool_Error_UnknownPeriodID(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "journal",
		"transactionDate": "2026-01-15",
		"description": "Test unknown period",
		"currency": "IDR",
		"periodId": 999,
		"totalAmount": "100000.00",
		"entries": [
			{
				"accountId": 101,
				"direction": "DEBIT",
				"amount": 100000
			},
			{
				"accountId": 102,
				"direction": "CREDIT",
				"amount": 100000
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_unknown_period",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Contains(t, res, "period 999 is not an active period in workspace")
	assert.Equal(t, 0, mockCtx.saveDraftCalls)
}

// Error case: Invalid transaction date format.
func TestExecuteTool_Error_InvalidDateFormat(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	args := `{
		"kind": "expense",
		"transactionDate": "15/01/2026",
		"description": "Bad date format",
		"currency": "IDR",
		"paymentAccountId": 101,
		"totalAmount": "50000.00",
		"items": [
			{
				"accountId": 201,
				"quantity": 1,
				"unitPrice": 50000
			}
		]
	}`

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_bad_date",
		Name:      agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(args),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	assert.Contains(t, res, "invalid transaction date")
	assert.Equal(t, 0, mockCtx.saveDraftCalls)
}

// Error case: Malformed / unparseable JSON argument handling.
func TestExecuteTool_Error_MalformedJSONArguments(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	testCases := []struct {
		name string
		args []byte
	}{
		{"UnclosedJSON", []byte(`{"kind": "expense",`)},
		{"NotJSON", []byte(`plain text input`)},
		{"EmptyString", []byte(``)},
		{"NilArguments", nil},
		{"InvalidItemAccountType", []byte(`{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Bad account type",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": [{"accountId": "not-an-id", "quantity": 1, "unitPrice": 50000}]
		}`)},
		{"InvalidQuantityType", []byte(`{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Bad quantity",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": [{"accountId": 201, "quantity": "abc", "unitPrice": 50000}]
		}`)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
					ID:        "call_malformed",
					Name:      agentsvc.ToolProposeDraft,
					Arguments: json.RawMessage(tc.args),
				})
				require.NoError(t, err)
				assert.Nil(t, proposal)
				assert.Contains(t, res, `"error"`)
			}, "must never panic on malformed arguments")
		})
	}

	assert.Equal(t, 0, mockCtx.saveDraftCalls)
}

// Error case: Unknown tool name handling.
func TestExecuteTool_Error_UnknownTool(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:        "call_unknown_tool",
		Name:      "some_unsupported_tool",
		Arguments: json.RawMessage(`{}`),
	})

	require.NoError(t, err)
	assert.Nil(t, proposal)
	var errResp agentsvc.ToolErrorResponse
	err = json.Unmarshal([]byte(res), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, `unknown tool: "some_unsupported_tool"`)
	assert.Equal(t, 0, mockCtx.saveDraftCalls)
}

// Additional validation edge cases.
func TestExecuteTool_ValidationEdgeCases(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	t.Run("ContextCanceled", func(t *testing.T) {
		cancCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, _, err := agentsvc.ExecuteTool(cancCtx, mockCtx, llmx.ToolCall{
			Name: agentsvc.ToolListAccounts,
		})
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("NilDraftContext", func(t *testing.T) {
		_, _, err := agentsvc.ExecuteTool(ctx, nil, llmx.ToolCall{
			Name: agentsvc.ToolListAccounts,
		})
		assert.Error(t, err)
	})

	t.Run("ForeignCurrencyWithExchangeRate", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Software subscription in USD",
			"currency": "USD",
			"exchangeRate": "16000.00",
			"paymentAccountId": 101,
			"totalAmount": "10.00",
			"items": [
				{"accountId": 201, "quantity": 1, "unitPrice": "10.00"}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_usd",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		require.NotNil(t, proposal)
		assert.Equal(t, "USD", proposal.Currency)
		require.NotNil(t, proposal.ExchangeRate)
		assert.True(t, proposal.ExchangeRate.Equal(decimal.NewFromInt(16000)))
		assert.Contains(t, res, `"status":"proposed"`)
	})

	t.Run("EmptyItemsInExpense", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "No items",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": []
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_no_items",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "at least 1 item is required")
	})

	t.Run("NegativeUnitPrice", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Negative unit price",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": [
				{"accountId": 201, "quantity": 1, "unitPrice": -50000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_neg_price",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "unit price cannot be negative")
	})

	t.Run("ZeroQuantity", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Zero quantity",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": [
				{"accountId": 201, "quantity": 0, "unitPrice": 50000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_zero_qty",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "quantity must be greater than zero")
	})

	t.Run("JournalLessThanTwoEntries", func(t *testing.T) {
		args := `{
			"kind": "journal",
			"transactionDate": "2026-01-15",
			"description": "One entry only",
			"currency": "IDR",
			"totalAmount": "100000.00",
			"entries": [
				{"accountId": 101, "direction": "DEBIT", "amount": 100000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_one_entry",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "journal draft requires at least 2 entries")
	})

	t.Run("JournalInvalidDirection", func(t *testing.T) {
		args := `{
			"kind": "journal",
			"transactionDate": "2026-01-15",
			"description": "Invalid direction",
			"currency": "IDR",
			"totalAmount": "100000.00",
			"entries": [
				{"accountId": 101, "direction": "SIDEWAYS", "amount": 100000},
				{"accountId": 102, "direction": "CREDIT", "amount": 100000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_bad_dir",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "invalid direction")
	})

	t.Run("JournalZeroAmount", func(t *testing.T) {
		args := `{
			"kind": "journal",
			"transactionDate": "2026-01-15",
			"description": "Zero entry amount",
			"currency": "IDR",
			"totalAmount": "100000.00",
			"entries": [
				{"accountId": 101, "direction": "DEBIT", "amount": 0},
				{"accountId": 102, "direction": "CREDIT", "amount": 100000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_zero_entry",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "amount must be greater than zero")
	})

	t.Run("MissingDescription", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "   ",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "50000.00",
			"items": [
				{"accountId": 201, "quantity": 1, "unitPrice": 50000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_no_desc",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "description is required")
	})

	t.Run("MissingPaymentAccountInExpense", func(t *testing.T) {
		args := `{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "No payment account",
			"currency": "IDR",
			"totalAmount": "50000.00",
			"items": [
				{"accountId": 201, "quantity": 1, "unitPrice": 50000}
			]
		}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_no_pmt",
			Name:      agentsvc.ToolProposeDraft,
			Arguments: json.RawMessage(args),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "paymentAccountId is required")
	})
}

// Proof of no-write: Assert that SaveDraft is never invoked during any tool execution.
func TestExecuteTool_ProofOfNoWrite(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	// Execute list_accounts
	_, _, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:   "call_list_acc",
		Name: agentsvc.ToolListAccounts,
	})
	require.NoError(t, err)

	// Execute list_currencies
	_, _, err = agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:   "call_list_curr",
		Name: agentsvc.ToolListCurrencies,
	})
	require.NoError(t, err)

	// Execute list_periods
	_, _, err = agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:   "call_list_per",
		Name: agentsvc.ToolListPeriods,
	})
	require.NoError(t, err)

	// Execute valid propose_draft (Reference Case 1)
	_, prop1, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:   "call_ref1",
		Name: agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(`{
			"kind": "journal",
			"transactionDate": "2026-01-15",
			"description": "Tarik tunai ATM",
			"currency": "IDR",
			"totalAmount": "2000000.00",
			"entries": [
				{"accountId": 101, "direction": "DEBIT", "amount": "2000000.00"},
				{"accountId": 102, "direction": "CREDIT", "amount": "2000000.00"}
			]
		}`),
	})
	require.NoError(t, err)
	require.NotNil(t, prop1)

	// Execute valid propose_draft (Reference Case 2)
	_, prop2, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:   "call_ref2",
		Name: agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(`{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"description": "Shop receipt",
			"currency": "IDR",
			"paymentAccountId": 101,
			"totalAmount": "70000.00",
			"items": [
				{"accountId": 201, "quantity": 1, "unitPrice": 70000}
			]
		}`),
	})
	require.NoError(t, err)
	require.NotNil(t, prop2)

	// Execute invalid propose_draft
	_, propBad, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
		ID:   "call_bad",
		Name: agentsvc.ToolProposeDraft,
		Arguments: json.RawMessage(`{
			"kind": "expense",
			"transactionDate": "2026-01-20",
			"paymentAccountId": 99999,
			"currency": "IDR",
			"items": []
		}`),
	})
	require.NoError(t, err)
	assert.Nil(t, propBad)

	// Zero-write assertion
	assert.Equal(t, 0, mockCtx.saveDraftCalls, "SaveDraft must NEVER be called by any tool execution!")
}

func TestExecuteTool_GetAccountBalances(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.accountBalancesRes = agentsvc.AccountBalanceResult{
		AsOfDate:        "2026-02-15",
		BaseCurrency:    "IDR",
		MatchedNetTotal: decimal.NewFromInt(15000000),
		TotalDebit:      decimal.NewFromInt(15000000),
		TotalCredit:     decimal.Zero,
		IsBalanced:      false,
		Accounts: []agentsvc.AccountBalanceItem{
			{
				AccountID:  101,
				Code:       "1101",
				Name:       "Kas Utama",
				Type:       "ASSET",
				NetBalance: decimal.NewFromInt(15000000),
			},
		},
	}

	t.Run("ValidQuery", func(t *testing.T) {
		rawArgs := `{"asOfDate":"2026-02-15","account":"1101","type":"ASSET","includePerCurrencyBreakdown":false}`
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_bal",
			Name:      agentsvc.ToolGetAccountBalances,
			Arguments: json.RawMessage(rawArgs),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.AccountBalanceResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.Equal(t, "IDR", out.BaseCurrency)
		assert.True(t, out.MatchedNetTotal.Equal(decimal.NewFromInt(15000000)))
		require.Len(t, out.Accounts, 1)
		assert.Equal(t, "Kas Utama", out.Accounts[0].Name)
	})

	t.Run("EmptyArgumentsUsesDefaults", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_bal_empty",
			Name:      agentsvc.ToolGetAccountBalances,
			Arguments: json.RawMessage(`{}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "Kas Utama")
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_bal_bad",
			Name:      agentsvc.ToolGetAccountBalances,
			Arguments: json.RawMessage(`{invalid`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "invalid arguments")
	})

	t.Run("ContextCancellationPropagates", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err := agentsvc.ExecuteTool(canceledCtx, mockCtx, llmx.ToolCall{
			ID:   "call_canceled",
			Name: agentsvc.ToolGetAccountBalances,
		})
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestExecuteTool_GetFinancialSummary(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.financialSummaryRes = agentsvc.FinancialSummaryResult{
		BaseCurrency: "IDR",
		Totals: agentsvc.FinancialSummaryTotals{
			Assets:      decimal.NewFromInt(50000000),
			Liabilities: decimal.NewFromInt(10000000),
			Equity:      decimal.NewFromInt(40000000),
			Revenue:     decimal.NewFromInt(25000000),
			Expense:     decimal.NewFromInt(15000000),
			NetIncome:   decimal.NewFromInt(10000000),
		},
		Monthly: []agentsvc.MonthlyRevenueExpense{
			{
				Month:   "2026-01",
				Revenue: decimal.NewFromInt(25000000),
				Expense: decimal.NewFromInt(15000000),
				Net:     decimal.NewFromInt(10000000),
			},
		},
	}

	t.Run("ValidQuery", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_summary",
			Name:      agentsvc.ToolGetFinancialSummary,
			Arguments: json.RawMessage(`{"fromMonth":"2026-01","toMonth":"2026-01","comparePrevious":true}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.FinancialSummaryResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.Equal(t, "IDR", out.BaseCurrency)
		assert.True(t, out.Totals.NetIncome.Equal(decimal.NewFromInt(10000000)))
		require.Len(t, out.Monthly, 1)
		assert.Equal(t, "2026-01", out.Monthly[0].Month)
	})
}

func TestExecuteTool_GetProfitLossBreakdown(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.profitLossRes = agentsvc.ProfitLossResult{
		DateFrom:     "2026-01-01",
		DateTo:       "2026-01-31",
		BaseCurrency: "IDR",
		TotalRevenue: decimal.NewFromInt(30000000),
		TotalExpense: decimal.NewFromInt(12000000),
		NetProfit:    decimal.NewFromInt(18000000),
		Revenues: []agentsvc.ProfitLossCategoryItem{
			{
				AccountID:   301,
				AccountCode: "4101",
				AccountName: "Pendapatan Operasional",
				Type:        "REVENUE",
				Amount:      decimal.NewFromInt(30000000),
			},
		},
		Expenses: []agentsvc.ProfitLossCategoryItem{
			{
				AccountID:   201,
				AccountCode: "5101",
				AccountName: "Beban Operasional",
				Type:        "EXPENSE",
				Amount:      decimal.NewFromInt(12000000),
			},
		},
	}

	t.Run("ValidQuery", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_pl",
			Name:      agentsvc.ToolGetProfitLossBreakdown,
			Arguments: json.RawMessage(`{"dateFrom":"2026-01-01","dateTo":"2026-01-31"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.ProfitLossResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.Equal(t, "IDR", out.BaseCurrency)
		assert.True(t, out.NetProfit.Equal(decimal.NewFromInt(18000000)))
		assert.True(t, out.TotalRevenue.Equal(decimal.NewFromInt(30000000)))
		assert.True(t, out.TotalExpense.Equal(decimal.NewFromInt(12000000)))
	})
}

func TestExecuteTool_SearchCashflows(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.cashflowsRes = agentsvc.CashflowSearchResult{
		ReturnedBaseAmountSum: decimal.NewFromInt(450000),
		Items: []agentsvc.CashflowDocItem{
			{
				ID:                 1,
				Number:             "EXP-2026-001",
				Date:               "2026-01-15",
				Type:               "expense",
				Status:             "posted",
				PaymentAccountCode: "1101",
				PaymentAccountName: "Kas Utama",
				Description:        "Beli ATK",
				Amount:             decimal.NewFromInt(450000),
				Currency:           "IDR",
				BaseAmount:         decimal.NewFromInt(450000),
			},
		},
	}

	t.Run("ValidExpenseSearch", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_cf",
			Name:      agentsvc.ToolSearchCashflows,
			Arguments: json.RawMessage(`{"type":"expense","dateFrom":"2026-01-01","limit":10}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.CashflowSearchResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.True(t, out.ReturnedBaseAmountSum.Equal(decimal.NewFromInt(450000)))
		require.Len(t, out.Items, 1)
		assert.Equal(t, "EXP-2026-001", out.Items[0].Number)
	})

	t.Run("MissingTypeReturnsError", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_cf_no_type",
			Name:      agentsvc.ToolSearchCashflows,
			Arguments: json.RawMessage(`{"dateFrom":"2026-01-01"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "type is required")
	})

	t.Run("InvalidTypeReturnsError", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_cf_bad_type",
			Name:      agentsvc.ToolSearchCashflows,
			Arguments: json.RawMessage(`{"type":"investment"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "must be 'income' or 'expense'")
	})

	t.Run("InvalidLimitReturnsError", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_cf_bad_lim",
			Name:      agentsvc.ToolSearchCashflows,
			Arguments: json.RawMessage(`{"type":"expense","limit":"ten"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "invalid limit: expected integer")
	})

	t.Run("ExplicitNullLimitAllowed", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_cf_null_lim",
			Name:      agentsvc.ToolSearchCashflows,
			Arguments: json.RawMessage(`{"type":"expense","limit":null}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.CashflowSearchResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.True(t, out.ReturnedBaseAmountSum.Equal(decimal.NewFromInt(450000)))
	})
}

func TestExecuteTool_GetAccountLedger(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.ledgerRes = agentsvc.AccountLedgerResult{
		AccountID:          101,
		AccountCode:        "1101",
		AccountName:        "Kas Utama",
		Type:               "ASSET",
		Currency:           "IDR",
		BaseCurrency:       "IDR",
		TotalCount:         1,
		Page:               1,
		Limit:              20,
		Postings: []agentsvc.AccountLedgerItem{
			{
				PostingID:          1001,
				EntryID:            501,
				EntryNumber:        "TRX-001",
				EntryDate:          "2026-01-10",
				Direction:          "DEBIT",
				Amount:             decimal.NewFromInt(5000000),
				BaseAmount:         decimal.NewFromInt(5000000),
				RunningBalance:     decimal.NewFromInt(5000000),
				RunningBaseBalance: decimal.NewFromInt(5000000),
			},
		},
	}

	t.Run("ValidAccountQuery", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_ledger",
			Name:      agentsvc.ToolGetAccountLedger,
			Arguments: json.RawMessage(`{"account":"1101","page":1,"limit":20}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.AccountLedgerResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.Equal(t, "1101", out.AccountCode)
		require.Len(t, out.Postings, 1)
		assert.True(t, out.Postings[0].RunningBalance.Equal(decimal.NewFromInt(5000000)))
	})

	t.Run("MissingAccountReturnsError", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_ledger_no_acc",
			Name:      agentsvc.ToolGetAccountLedger,
			Arguments: json.RawMessage(`{"page":1}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "account is required")
	})

	t.Run("AmbiguousMatchesReturnedInPayload", func(t *testing.T) {
		ambigCtx := newTestMockContext()
		ambigCtx.ledgerRes = agentsvc.AccountLedgerResult{
			AmbiguousMatches: []agentsvc.AccountOption{
				{ID: 101, Code: "1101", Name: "Kas Utama"},
				{ID: 102, Code: "1102", Name: "Kas Kecil"},
			},
		}
		res, proposal, err := agentsvc.ExecuteTool(ctx, ambigCtx, llmx.ToolCall{
			ID:        "call_ledger_ambig",
			Name:      agentsvc.ToolGetAccountLedger,
			Arguments: json.RawMessage(`{"account":"Kas"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.AccountLedgerResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		require.Len(t, out.AmbiguousMatches, 2)
		assert.Equal(t, "1101", out.AmbiguousMatches[0].Code)
		assert.Equal(t, "1102", out.AmbiguousMatches[1].Code)
	})

	t.Run("ExplicitNullPageAndLimitAllowed", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_ledger_null_params",
			Name:      agentsvc.ToolGetAccountLedger,
			Arguments: json.RawMessage(`{"account":"1101","page":null,"limit":null}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.AccountLedgerResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.Equal(t, "1101", out.AccountCode)
	})
}

func TestExecuteTool_SearchTransactions(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.transactionsRes = agentsvc.TransactionSearchResult{
		Items: []agentsvc.TransactionItem{
			{
				ID:              501,
				Number:          "TRX-2026-0001",
				TransactionDate: "2026-01-10",
				Description:     "Setoran Modal",
				Currency:        "IDR",
				ExchangeRate:    decimal.NewFromInt(1),
				Source:          "MANUAL",
				EntryCount:      2,
			},
		},
	}

	t.Run("ValidTransactionSearch", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_trx",
			Name:      agentsvc.ToolSearchTransactions,
			Arguments: json.RawMessage(`{"query":"Setoran","source":"MANUAL","limit":10}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.TransactionSearchResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		require.Len(t, out.Items, 1)
		assert.Equal(t, "TRX-2026-0001", out.Items[0].Number)
		assert.Equal(t, 2, out.Items[0].EntryCount)
	})

	t.Run("ExplicitNullLimitAllowed", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_trx_null_lim",
			Name:      agentsvc.ToolSearchTransactions,
			Arguments: json.RawMessage(`{"query":"Setoran","source":"MANUAL","limit":null}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.TransactionSearchResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		require.Len(t, out.Items, 1)
		assert.Equal(t, "TRX-2026-0001", out.Items[0].Number)
	})
}

func TestExecuteTool_LookupExchangeRate(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()
	mockCtx.exchangeRateRes = agentsvc.ExchangeRateResult{
		FromCurrency: "USD",
		ToCurrency:   "IDR",
		Date:         "2026-02-15",
		Rate:         decimal.NewFromInt(16250),
	}

	t.Run("ValidLookup", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_rate",
			Name:      agentsvc.ToolLookupExchangeRate,
			Arguments: json.RawMessage(`{"fromCurrency":"usd","toCurrency":"idr","date":"2026-02-15"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)

		var out agentsvc.ExchangeRateResult
		err = json.Unmarshal([]byte(res), &out)
		require.NoError(t, err)
		assert.Equal(t, "USD", out.FromCurrency)
		assert.Equal(t, "IDR", out.ToCurrency)
		assert.True(t, out.Rate.Equal(decimal.NewFromInt(16250)))
	})

	t.Run("MissingCurrenciesReturnsError", func(t *testing.T) {
		res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, llmx.ToolCall{
			ID:        "call_rate_no_curr",
			Name:      agentsvc.ToolLookupExchangeRate,
			Arguments: json.RawMessage(`{"fromCurrency":"USD"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, proposal)
		assert.Contains(t, res, "toCurrency is required")
	})
}

func TestExecuteTool_ReadonlyToolsProofOfNoProposal(t *testing.T) {
	ctx := context.Background()
	mockCtx := newTestMockContext()

	readOnlyTools := []llmx.ToolCall{
		{Name: agentsvc.ToolGetAccountBalances, Arguments: json.RawMessage(`{}`)},
		{Name: agentsvc.ToolGetFinancialSummary, Arguments: json.RawMessage(`{}`)},
		{Name: agentsvc.ToolGetProfitLossBreakdown, Arguments: json.RawMessage(`{}`)},
		{Name: agentsvc.ToolSearchCashflows, Arguments: json.RawMessage(`{"type":"expense"}`)},
		{Name: agentsvc.ToolGetAccountLedger, Arguments: json.RawMessage(`{"account":"1101"}`)},
		{Name: agentsvc.ToolSearchTransactions, Arguments: json.RawMessage(`{}`)},
		{Name: agentsvc.ToolLookupExchangeRate, Arguments: json.RawMessage(`{"fromCurrency":"USD","toCurrency":"IDR"}`)},
	}

	for _, call := range readOnlyTools {
		t.Run(call.Name, func(t *testing.T) {
			res, proposal, err := agentsvc.ExecuteTool(ctx, mockCtx, call)
			require.NoError(t, err)
			assert.Nil(t, proposal, "read-only tool %s must never return a DraftProposal", call.Name)
			assert.NotEmpty(t, res)
		})
	}
}
