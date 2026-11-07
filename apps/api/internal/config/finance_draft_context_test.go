package config

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	agentsvc "github.com/kilip/omed/api/internal/agent/service"
	findomain "github.com/kilip/omed/api/internal/fin/domain"
	finmodel "github.com/kilip/omed/api/internal/fin/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDraftCashFlowService struct {
	lastExpenseReq *finmodel.SaveCashFlowRequest
	lastIncomeReq  *finmodel.SaveCashFlowRequest
	expenseItem    findomain.CashFlowItem
	incomeItem     findomain.CashFlowItem
	expenseErr     error
	incomeErr      error
}

func (f *fakeDraftCashFlowService) CreateExpense(c context.Context, req finmodel.SaveCashFlowRequest) (findomain.CashFlowItem, error) {
	f.lastExpenseReq = &req
	if f.expenseErr != nil {
		return findomain.CashFlowItem{}, f.expenseErr
	}
	return f.expenseItem, nil
}

func (f *fakeDraftCashFlowService) CreateIncome(c context.Context, req finmodel.SaveCashFlowRequest) (findomain.CashFlowItem, error) {
	f.lastIncomeReq = &req
	if f.incomeErr != nil {
		return findomain.CashFlowItem{}, f.incomeErr
	}
	return f.incomeItem, nil
}

type fakeDraftTransactionService struct {
	lastReq       *finmodel.CreateTransactionRequest
	tx            findomain.Transaction
	err           error
	searchReq     finmodel.SearchTransactionRequest
	searchRes     []findomain.Transaction
	searchCursor  string
	searchErr     error
}

func (f *fakeDraftTransactionService) Create(c context.Context, req finmodel.CreateTransactionRequest) (findomain.Transaction, error) {
	f.lastReq = &req
	if f.err != nil {
		return findomain.Transaction{}, f.err
	}
	return f.tx, nil
}

func (f *fakeDraftTransactionService) Search(c context.Context, req finmodel.SearchTransactionRequest) ([]findomain.Transaction, string, error) {
	f.searchReq = req
	if f.searchErr != nil {
		return nil, "", f.searchErr
	}
	return f.searchRes, f.searchCursor, nil
}

type fakeDraftInsightService struct {
	balanceReq  finmodel.AccountBalanceRequest
	balanceRes  finmodel.AccountBalanceResponse
	balanceErr  error
	overviewReq finmodel.PeriodOverviewRequest
	overviewRes finmodel.PeriodOverviewResponse
	overviewErr error
	cashflowReq finmodel.FindCashflowRequest
	cashflowRes finmodel.FindCashflowResponse
	cashflowErr error
}

func (f *fakeDraftInsightService) AccountBalance(ctx context.Context, req finmodel.AccountBalanceRequest) (finmodel.AccountBalanceResponse, error) {
	f.balanceReq = req
	if f.balanceErr != nil {
		return finmodel.AccountBalanceResponse{}, f.balanceErr
	}
	return f.balanceRes, nil
}

func (f *fakeDraftInsightService) PeriodOverview(c context.Context, req finmodel.PeriodOverviewRequest) (finmodel.PeriodOverviewResponse, error) {
	f.overviewReq = req
	if f.overviewErr != nil {
		return finmodel.PeriodOverviewResponse{}, f.overviewErr
	}
	return f.overviewRes, nil
}

func (f *fakeDraftInsightService) FindCashflow(c context.Context, req finmodel.FindCashflowRequest) (finmodel.FindCashflowResponse, error) {
	f.cashflowReq = req
	if f.cashflowErr != nil {
		return finmodel.FindCashflowResponse{}, f.cashflowErr
	}
	return f.cashflowRes, nil
}

type fakeDraftReportService struct {
	baseCurrency string
	tbReq        finmodel.TrialBalanceRequest
	tbRes        finmodel.TrialBalanceResponse
	tbErr        error
	summaryRes   finmodel.FinancialSummaryResponse
	summaryErr   error
	plReq        finmodel.ProfitLossBreakdownRequest
	plRes        finmodel.ProfitLossBreakdownResponse
	plErr        error
}

func (f *fakeDraftReportService) BaseCurrency(ctx context.Context) string {
	if f.baseCurrency != "" {
		return f.baseCurrency
	}
	return "IDR"
}

func (f *fakeDraftReportService) TrialBalance(ctx context.Context, req finmodel.TrialBalanceRequest) (finmodel.TrialBalanceResponse, error) {
	f.tbReq = req
	if f.tbErr != nil {
		return finmodel.TrialBalanceResponse{}, f.tbErr
	}
	return f.tbRes, nil
}

func (f *fakeDraftReportService) Summary(ctx context.Context, months int) (finmodel.FinancialSummaryResponse, error) {
	if f.summaryErr != nil {
		return finmodel.FinancialSummaryResponse{}, f.summaryErr
	}
	return f.summaryRes, nil
}

func (f *fakeDraftReportService) ProfitLossBreakdown(ctx context.Context, req finmodel.ProfitLossBreakdownRequest) (finmodel.ProfitLossBreakdownResponse, error) {
	f.plReq = req
	if f.plErr != nil {
		return finmodel.ProfitLossBreakdownResponse{}, f.plErr
	}
	return f.plRes, nil
}

type fakeDraftEntryService struct {
	ledgerAccountID int64
	ledgerPage      int
	ledgerLimit     int
	postings        []findomain.Posting
	ledgerTotal     int64
	ledgerErr       error
	entries         map[int64]findomain.Entry
	getByIDError    error
}

func (f *fakeDraftEntryService) GetAccountLedger(c context.Context, accountID int64, page int, limit int) ([]findomain.Posting, int64, error) {
	f.ledgerAccountID = accountID
	f.ledgerPage = page
	f.ledgerLimit = limit
	if f.ledgerErr != nil {
		return nil, 0, f.ledgerErr
	}
	return f.postings, f.ledgerTotal, nil
}

func (f *fakeDraftEntryService) GetByID(c context.Context, id int64) (findomain.Entry, error) {
	if f.getByIDError != nil {
		return findomain.Entry{}, f.getByIDError
	}
	if e, ok := f.entries[id]; ok {
		return e, nil
	}
	return findomain.Entry{}, nil
}

type fakeDraftExchangeRateService struct {
	lookupFrom string
	lookupTo   string
	lookupDate time.Time
	lookupRate decimal.Decimal
	lookupErr  error
	searchReq  finmodel.SearchExchangeRateRequest
	searchRes  []findomain.ExchangeRate
	searchErr  error
}

func (f *fakeDraftExchangeRateService) Lookup(c context.Context, from, to string, date time.Time) (decimal.Decimal, error) {
	f.lookupFrom = from
	f.lookupTo = to
	f.lookupDate = date
	if f.lookupErr != nil {
		return decimal.Zero, f.lookupErr
	}
	return f.lookupRate, nil
}

func (f *fakeDraftExchangeRateService) Search(c context.Context, req finmodel.SearchExchangeRateRequest) ([]findomain.ExchangeRate, string, error) {
	f.searchReq = req
	if f.searchErr != nil {
		return nil, "", f.searchErr
	}
	return f.searchRes, "", nil
}
func TestFinanceDraftContext_ListAccounts(t *testing.T) {
	ctx := context.Background()

	accounts := &fakeAccountSearcher{
		accounts: []findomain.Account{
			{ID: 1, Code: "1101", Name: "Cash on Hand", IsActive: true, IsPostable: true},
			{ID: 2, Code: "1102", Name: "Bank BCA", IsActive: true, IsPostable: true},
			{ID: 3, Code: "1100", Name: "Cash & Bank Header", IsActive: true, IsPostable: false}, // non-postable
			{ID: 4, Code: "1103", Name: "Old Cash", IsActive: false, IsPostable: true},            // inactive
		},
	}

	adapter := newFinanceDraftContextFromServices(accounts, nil, nil, nil, nil, nil, nil, nil, nil)
	opts, err := adapter.ListAccounts(ctx)
	require.NoError(t, err)
	require.Len(t, opts, 2)
	assert.Equal(t, int64(1), opts[0].ID)
	assert.Equal(t, "1101", opts[0].Code)
	assert.Equal(t, "Cash on Hand", opts[0].Name)
	assert.Equal(t, int64(2), opts[1].ID)
	assert.Equal(t, "1102", opts[1].Code)
}

func TestFinanceDraftContext_ListCurrencies(t *testing.T) {
	ctx := context.Background()

	currencies := &fakeCurrencyLister{
		currencies: []findomain.Currency{
			{ID: 1, Code: "IDR", Name: "Indonesian Rupiah", Symbol: "Rp", Decimals: 0},
			{ID: 2, Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2},
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, currencies, nil, nil, nil, nil, nil, nil, nil)
	opts, err := adapter.ListCurrencies(ctx)
	require.NoError(t, err)
	require.Len(t, opts, 2)
	assert.Equal(t, "IDR", opts[0].Code)
	assert.Equal(t, "Indonesian Rupiah", opts[0].Name)
	assert.Equal(t, "Rp", opts[0].Symbol)
	assert.Equal(t, 0, opts[0].Decimals)
	assert.Equal(t, "USD", opts[1].Code)
	assert.Equal(t, 2, opts[1].Decimals)
}

func TestFinanceDraftContext_ListPeriods(t *testing.T) {
	ctx := context.Background()

	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	periods := &fakePeriodSearcher{
		periods: []findomain.Period{
			{
				ID:         10,
				Name:       "Jan 2026",
				StartDate:  startDate,
				EndDate:    endDate,
				FiscalYear: 2026,
				Status:     findomain.PeriodStatusOpen,
			},
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, periods, nil, nil, nil, nil, nil, nil)
	opts, err := adapter.ListPeriods(ctx)
	require.NoError(t, err)
	require.Len(t, opts, 1)
	assert.Equal(t, int64(10), opts[0].ID)
	assert.Equal(t, "Jan 2026", opts[0].Name)
	assert.Equal(t, 2026, opts[0].FiscalYear)
	assert.Equal(t, findomain.PeriodStatusOpen, opts[0].Status)
	require.NotNil(t, periods.lastReq)
	require.NotNil(t, periods.lastReq.Status)
	assert.Equal(t, findomain.PeriodStatusOpen, *periods.lastReq.Status)
}

func TestFinanceDraftContext_SaveDraft_Expense_ThreeItems(t *testing.T) {
	ctx := context.Background()

	cashFlows := &fakeDraftCashFlowService{
		expenseItem: findomain.CashFlowItem{
			ID:          42,
			EntryStatus: findomain.EntryStatusDraft,
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, cashFlows, nil, nil, nil, nil, nil)

	paymentAccID := int64(101)
	proposal := model.DraftProposal{
		Kind:             model.DraftKindExpense,
		TransactionDate:  "2026-01-15",
		Description:      "Warung shop supplies",
		Reference:        "RCP-001",
		Currency:         "IDR",
		PaymentAccountID: &paymentAccID,
		TotalAmount:      decimal.NewFromInt(70000),
		Items: []model.DraftItem{
			{
				AccountID:   5001,
				Description: "Notebook",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.NewFromInt(15000),
				Amount:      decimal.NewFromInt(30000),
			},
			{
				AccountID:   5002,
				Description: "Pen set",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(25000),
				Amount:      decimal.NewFromInt(25000),
			},
			{
				AccountID:   5003,
				Description: "Stapler",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(15000),
				Amount:      decimal.NewFromInt(15000),
			},
		},
	}

	id, err := adapter.SaveDraft(ctx, proposal)
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)

	require.NotNil(t, cashFlows.lastExpenseReq)
	req := cashFlows.lastExpenseReq
	assert.Equal(t, paymentAccID, req.PaymentAccountID)
	assert.Equal(t, findomain.EntryStatusDraft, req.Status)
	assert.Equal(t, "RCP-001", req.Reference)
	assert.Equal(t, "Warung shop supplies", req.Description)
	assert.Equal(t, "IDR", req.Currency)
	assert.Equal(t, "2026-01-15", req.TransactionDate.Time().Format("2006-01-02"))
	require.Len(t, req.Items, 3)

	assert.Equal(t, int64(5001), req.Items[0].AccountID)
	assert.Equal(t, "Notebook", req.Items[0].Description)
	assert.True(t, req.Items[0].Quantity.Equal(decimal.NewFromInt(2)))
	assert.True(t, req.Items[0].UnitPrice.Equal(decimal.NewFromInt(15000)))

	assert.Equal(t, int64(5002), req.Items[1].AccountID)
	assert.Equal(t, "Pen set", req.Items[1].Description)

	assert.Equal(t, int64(5003), req.Items[2].AccountID)
	assert.Equal(t, "Stapler", req.Items[2].Description)
}

func TestFinanceDraftContext_SaveDraft_Income(t *testing.T) {
	ctx := context.Background()

	cashFlows := &fakeDraftCashFlowService{
		incomeItem: findomain.CashFlowItem{
			ID:          99,
			EntryStatus: findomain.EntryStatusDraft,
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, cashFlows, nil, nil, nil, nil, nil)

	paymentAccID := int64(201)
	proposal := model.DraftProposal{
		Kind:             model.DraftKindIncome,
		TransactionDate:  "2026-02-01",
		Description:      "Consulting revenue invoice",
		Reference:        "INV-2026-01",
		Currency:         "IDR",
		PaymentAccountID: &paymentAccID,
		TotalAmount:      decimal.NewFromInt(5000000),
		Items: []model.DraftItem{
			{
				AccountID:   4001,
				Description: "Architecture review",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(5000000),
				Amount:      decimal.NewFromInt(5000000),
			},
		},
	}

	id, err := adapter.SaveDraft(ctx, proposal)
	require.NoError(t, err)
	assert.Equal(t, int64(99), id)

	require.NotNil(t, cashFlows.lastIncomeReq)
	req := cashFlows.lastIncomeReq
	assert.Equal(t, paymentAccID, req.PaymentAccountID)
	assert.Equal(t, findomain.EntryStatusDraft, req.Status)
	assert.Equal(t, "INV-2026-01", req.Reference)
	assert.Equal(t, "Consulting revenue invoice", req.Description)
	assert.Equal(t, "2026-02-01", req.TransactionDate.Time().Format("2006-01-02"))
	require.Len(t, req.Items, 1)
	assert.Equal(t, int64(4001), req.Items[0].AccountID)
}

func TestFinanceDraftContext_SaveDraft_Journal_BankKaltimtaraATMWithdrawal(t *testing.T) {
	ctx := context.Background()

	txs := &fakeDraftTransactionService{
		tx: findomain.Transaction{
			ID: 777,
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, txs, nil, nil, nil, nil)

	periodID := int64(5)
	amount := decimal.NewFromInt(2000000)

	proposal := model.DraftProposal{
		Kind:            model.DraftKindJournal,
		TransactionDate: "2026-01-20",
		Description:     "Bank Kaltimtara ATM withdrawal",
		Reference:       "ATM-KALTIM-001",
		Currency:        "IDR",
		PeriodID:        &periodID,
		TotalAmount:     amount,
		Entries: []model.DraftJournalEntry{
			{
				AccountID:   1101, // Cash on Hand
				Direction:   "DEBIT",
				Amount:      amount,
				Description: "Dr Cash on Hand",
			},
			{
				AccountID:   1102, // Bank Kaltimtara
				Direction:   "CREDIT",
				Amount:      amount,
				Description: "Cr Bank Kaltimtara",
			},
		},
	}

	id, err := adapter.SaveDraft(ctx, proposal)
	require.NoError(t, err)
	assert.Equal(t, int64(777), id)

	require.NotNil(t, txs.lastReq)
	req := txs.lastReq
	assert.Equal(t, "2026-01-20", req.TransactionDate.Time().Format("2006-01-02"))
	assert.Equal(t, "Bank Kaltimtara ATM withdrawal", req.Description)
	assert.Equal(t, "ATM-KALTIM-001", req.Reference)
	assert.Equal(t, "IDR", req.Currency)

	require.NotNil(t, req.Entry, "inline entry must be created")
	entry := req.Entry
	assert.Equal(t, periodID, entry.PeriodID)
	assert.Equal(t, "2026-01-20", entry.EntryDate.Time().Format("2006-01-02"))
	assert.Equal(t, "Bank Kaltimtara ATM withdrawal", entry.Description)
	require.Len(t, entry.Postings, 2)

	// Balanced entries check
	assert.Equal(t, int64(1101), entry.Postings[0].AccountID)
	assert.Equal(t, "DEBIT", entry.Postings[0].Direction)
	assert.True(t, entry.Postings[0].Amount.Equal(amount))

	assert.Equal(t, int64(1102), entry.Postings[1].AccountID)
	assert.Equal(t, "CREDIT", entry.Postings[1].Direction)
	assert.True(t, entry.Postings[1].Amount.Equal(amount))
}

func TestFinanceDraftContext_SaveDraft_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Unsupported kind
	_, err := adapter.SaveDraft(ctx, model.DraftProposal{Kind: "unknown"})
	assert.ErrorContains(t, err, "unsupported draft kind")

	// Expense missing payment account
	_, err = adapter.SaveDraft(ctx, model.DraftProposal{
		Kind:            model.DraftKindExpense,
		TransactionDate: "2026-01-01",
	})
	assert.ErrorContains(t, err, "cash flow service is not configured")

	cashFlows := &fakeDraftCashFlowService{}
	adapterWithCF := newFinanceDraftContextFromServices(nil, nil, nil, cashFlows, nil, nil, nil, nil, nil)

	_, err = adapterWithCF.SaveDraft(ctx, model.DraftProposal{
		Kind:            model.DraftKindExpense,
		TransactionDate: "2026-01-01",
	})
	assert.ErrorContains(t, err, "payment account is required for expense draft")

	// Income missing payment account
	_, err = adapterWithCF.SaveDraft(ctx, model.DraftProposal{
		Kind:            model.DraftKindIncome,
		TransactionDate: "2026-01-01",
	})
	assert.ErrorContains(t, err, "payment account is required for income draft")

	// Journal missing period ID
	txs := &fakeDraftTransactionService{}
	adapterWithTx := newFinanceDraftContextFromServices(nil, nil, nil, nil, txs, nil, nil, nil, nil)
	_, err = adapterWithTx.SaveDraft(ctx, model.DraftProposal{
		Kind:            model.DraftKindJournal,
		TransactionDate: "2026-01-01",
	})
	assert.ErrorContains(t, err, "period ID is required for journal draft")

	// Invalid date
	accID := int64(1)
	_, err = adapterWithCF.SaveDraft(ctx, model.DraftProposal{
		Kind:             model.DraftKindExpense,
		TransactionDate:  "invalid-date",
		PaymentAccountID: &accID,
	})
	assert.ErrorContains(t, err, "invalid transaction date")
}

func TestFinanceDraftContext_NilModule(t *testing.T) {
	assert.Nil(t, newFinanceDraftContext(nil))
}

func TestFinanceDraftContext_NilDependencies(t *testing.T) {
	ctx := context.Background()
	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	accs, err := adapter.ListAccounts(ctx)
	require.NoError(t, err)
	assert.Nil(t, accs)

	currs, err := adapter.ListCurrencies(ctx)
	require.NoError(t, err)
	assert.Nil(t, currs)

	periods, err := adapter.ListPeriods(ctx)
	require.NoError(t, err)
	assert.Nil(t, periods)
}
func TestFinanceDraftContext_GetAccountBalances(t *testing.T) {
	ctx := context.Background()

	// 1. Standard query delegated to InsightService
	insights := &fakeDraftInsightService{
		balanceRes: finmodel.AccountBalanceResponse{
			AsOfDate:     "2026-01-31",
			BaseCurrency: "IDR",
			Accounts: []finmodel.AccountBalanceItem{
				{Code: "1101", Name: "Kas", Type: "ASSET", Debit: decimal.NewFromInt(100), Credit: decimal.Zero, Net: decimal.NewFromInt(100)},
				{Code: "1102", Name: "Bank", Type: "ASSET", Debit: decimal.NewFromInt(200), Credit: decimal.Zero, Net: decimal.NewFromInt(200)},
			},
			Totals: finmodel.BalanceTotals{
				Debit:  decimal.NewFromInt(300),
				Credit: decimal.Zero,
			},
			IsBalanced: true,
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, insights, nil, nil, nil)
	res, err := adapter.GetAccountBalances(ctx, agentsvc.AccountBalanceQuery{
		AsOfDate: "2026-01-31",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-01-31", res.AsOfDate)
	assert.Equal(t, "IDR", res.BaseCurrency)
	assert.Len(t, res.Accounts, 2)
	assert.True(t, res.MatchedNetTotal.Equal(decimal.NewFromInt(300)))
	assert.True(t, res.TotalDebit.Equal(decimal.NewFromInt(300)))

	// 2. Query with IncludePerCurrencyBreakdown or Type filter using ReportService.TrialBalance
	reports := &fakeDraftReportService{
		baseCurrency: "IDR",
		tbRes: finmodel.TrialBalanceResponse{
			AsOfDate:     finmodel.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)),
			BaseCurrency: "IDR",
			Accounts: []finmodel.TrialBalanceAccountItem{
				{
					AccountID:   1,
					AccountCode: "1101",
					AccountName: "Kas",
					Type:        "ASSET",
					Debit:       decimal.NewFromInt(500),
					Credit:      decimal.Zero,
					NetBalance:  decimal.NewFromInt(500),
					Currencies: []finmodel.CurrencyBreakdownItem{
						{Currency: "USD", Debit: decimal.NewFromInt(30), NetBalance: decimal.NewFromInt(30)},
					},
				},
				{
					AccountID:   2,
					AccountCode: "2101",
					AccountName: "Hutang",
					Type:        "LIABILITY",
					Debit:       decimal.Zero,
					Credit:      decimal.NewFromInt(500),
					NetBalance:  decimal.NewFromInt(500),
				},
			},
			TotalDebit:  decimal.NewFromInt(500),
			TotalCredit: decimal.NewFromInt(500),
			IsBalanced:  true,
		},
	}

	adapterWithReports := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, nil, reports, nil, nil)
	tbRes, err := adapterWithReports.GetAccountBalances(ctx, agentsvc.AccountBalanceQuery{
		Type:                        "ASSET",
		IncludePerCurrencyBreakdown: true,
	})
	require.NoError(t, err)
	assert.Len(t, tbRes.Accounts, 1)
	assert.Equal(t, "1101", tbRes.Accounts[0].Code)
	assert.Len(t, tbRes.Accounts[0].Currencies, 1)
	assert.Equal(t, "USD", tbRes.Accounts[0].Currencies[0].Currency)
	assert.True(t, tbRes.MatchedNetTotal.Equal(decimal.NewFromInt(500)))
}

func TestFinanceDraftContext_GetFinancialSummary(t *testing.T) {
	ctx := context.Background()

	reports := &fakeDraftReportService{
		baseCurrency: "IDR",
		summaryRes: finmodel.FinancialSummaryResponse{
			BaseCurrency: "IDR",
			Totals: finmodel.FinancialSummaryTotals{
				Assets:      decimal.NewFromInt(100000000),
				Liabilities: decimal.NewFromInt(40000000),
				Equity:      decimal.NewFromInt(60000000),
				Revenue:     decimal.NewFromInt(30000000),
				Expense:     decimal.NewFromInt(10000000),
				NetIncome:   decimal.NewFromInt(20000000),
			},
			Monthly: []finmodel.MonthlyRevenueExpenseItem{
				{Month: "2026-01", Revenue: decimal.NewFromInt(30000000), Expense: decimal.NewFromInt(10000000)},
			},
		},
	}

	insights := &fakeDraftInsightService{
		overviewRes: finmodel.PeriodOverviewResponse{
			BaseCurrency: "IDR",
			From:         "2026-01",
			To:           "2026-02",
			Monthly: []finmodel.MonthTotals{
				{Month: "2026-01", Revenue: decimal.NewFromInt(30000000), Expense: decimal.NewFromInt(10000000), Net: decimal.NewFromInt(20000000)},
				{Month: "2026-02", Revenue: decimal.NewFromInt(40000000), Expense: decimal.NewFromInt(15000000), Net: decimal.NewFromInt(25000000)},
			},
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, insights, reports, nil, nil)
	res, err := adapter.GetFinancialSummary(ctx, agentsvc.FinancialSummaryQuery{
		FromMonth: "2026-01",
		ToMonth:   "2026-02",
	})
	require.NoError(t, err)
	assert.Equal(t, "IDR", res.BaseCurrency)
	assert.True(t, res.Totals.Assets.Equal(decimal.NewFromInt(100000000)))
	assert.True(t, res.Totals.NetIncome.Equal(decimal.NewFromInt(20000000)))
	assert.Len(t, res.Monthly, 2)
	assert.Equal(t, "2026-02", res.Monthly[1].Month)
}

func TestFinanceDraftContext_GetProfitLossBreakdown(t *testing.T) {
	ctx := context.Background()

	reports := &fakeDraftReportService{
		baseCurrency: "IDR",
		plRes: finmodel.ProfitLossBreakdownResponse{
			DateFrom:     finmodel.Date(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			DateTo:       finmodel.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)),
			BaseCurrency: "IDR",
			Revenues: []finmodel.ProfitLossCategoryItem{
				{AccountID: 10, AccountCode: "4101", AccountName: "Pendapatan Jasa", Type: "REVENUE", Amount: decimal.NewFromInt(5000000)},
			},
			Expenses: []finmodel.ProfitLossCategoryItem{
				{AccountID: 20, AccountCode: "5101", AccountName: "Beban Listrik", Type: "EXPENSE", Amount: decimal.NewFromInt(1000000)},
			},
			TotalRevenue: decimal.NewFromInt(5000000),
			TotalExpense: decimal.NewFromInt(1000000),
			NetProfit:    decimal.NewFromInt(4000000),
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, nil, reports, nil, nil)
	res, err := adapter.GetProfitLossBreakdown(ctx, agentsvc.ProfitLossQuery{
		DateFrom: "2026-01-01",
		DateTo:   "2026-01-31",
	})
	require.NoError(t, err)
	assert.Equal(t, "IDR", res.BaseCurrency)
	assert.Len(t, res.Revenues, 1)
	assert.Equal(t, "Pendapatan Jasa", res.Revenues[0].AccountName)
	assert.Len(t, res.Expenses, 1)
	assert.Equal(t, "Beban Listrik", res.Expenses[0].AccountName)
	assert.True(t, res.NetProfit.Equal(decimal.NewFromInt(4000000)))
}

func TestFinanceDraftContext_SearchCashflows(t *testing.T) {
	ctx := context.Background()

	insights := &fakeDraftInsightService{
		cashflowRes: finmodel.FindCashflowResponse{
			Items: []finmodel.CashflowDoc{
				{
					ID:             1,
					Number:         "EXP-001",
					Date:           "2026-01-10",
					Type:           "expense",
					Status:         "posted",
					PaymentAccount: finmodel.AccountRef{Code: "1101", Name: "Kas"},
					Amount:         decimal.NewFromInt(50000),
					Currency:       "IDR",
					BaseAmount:     decimal.NewFromInt(50000),
					Lines: []finmodel.CashflowLine{
						{
							Account:     finmodel.AccountRef{Code: "5101", Name: "ATK"},
							Description: "Kertas",
							Quantity:    decimal.NewFromInt(1),
							UnitPrice:   decimal.NewFromInt(50000),
							Amount:      decimal.NewFromInt(50000),
						},
					},
				},
				{
					ID:             2,
					Number:         "EXP-002",
					Date:           "2026-01-11",
					Type:           "expense",
					Status:         "posted",
					PaymentAccount: finmodel.AccountRef{Code: "1101", Name: "Kas"},
					Amount:         decimal.NewFromInt(25000),
					Currency:       "IDR",
					BaseAmount:     decimal.NewFromInt(25000),
				},
			},
		},
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, insights, nil, nil, nil)
	res, err := adapter.SearchCashflows(ctx, agentsvc.CashflowSearchQuery{
		Type: "expense",
	})
	require.NoError(t, err)
	assert.Len(t, res.Items, 2)
	assert.Equal(t, "EXP-001", res.Items[0].Number)
	assert.Len(t, res.Items[0].Lines, 1)
	assert.Equal(t, "Kertas", res.Items[0].Lines[0].Description)
	assert.True(t, res.ReturnedBaseAmountSum.Equal(decimal.NewFromInt(75000)))
}

func TestFinanceDraftContext_GetAccountLedger_AmbiguityAndSuccess(t *testing.T) {
	ctx := context.Background()

	accounts := &fakeAccountSearcher{
		accounts: []findomain.Account{
			{ID: 101, Code: "1101", Name: "Bank Kaltim IDR", Currency: "IDR", Type: "ASSET"},
			{ID: 102, Code: "1102", Name: "Bank Kaltim USD", Currency: "USD", Type: "ASSET"},
			{ID: 103, Code: "1103", Name: "Petty Cash", Currency: "IDR", Type: "ASSET"},
		},
	}

	entries := &fakeDraftEntryService{
		postings: []findomain.Posting{
			{
				ID:                 1,
				EntryID:            10,
				Direction:          "DEBIT",
				Amount:             decimal.NewFromInt(1000),
				BaseAmount:         decimal.NewFromInt(16000000),
				RunningBalance:     decimal.NewFromInt(1000),
				RunningBaseBalance: decimal.NewFromInt(16000000),
				Description:        "Initial deposit",
			},
		},
		ledgerTotal: 1,
		entries: map[int64]findomain.Entry{
			10: {
				ID:        10,
				Number:    "JE-2026-0001",
				EntryDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	adapter := newFinanceDraftContextFromServices(accounts, nil, nil, nil, nil, nil, nil, entries, nil)

	// 1. Ambiguous query "Bank Kaltim" matches both 101 and 102
	ambRes, err := adapter.GetAccountLedger(ctx, agentsvc.AccountLedgerQuery{
		Account: "Bank Kaltim",
	})
	require.NoError(t, err)
	assert.Len(t, ambRes.AmbiguousMatches, 2)
	assert.Empty(t, ambRes.Postings)

	// 2. Exact code match "1102" resolves to single USD account
	res, err := adapter.GetAccountLedger(ctx, agentsvc.AccountLedgerQuery{
		Account: "1102",
	})
	require.NoError(t, err)
	assert.Empty(t, res.AmbiguousMatches)
	assert.Equal(t, int64(102), res.AccountID)
	assert.Equal(t, "1102", res.AccountCode)
	assert.Equal(t, "USD", res.Currency)
	assert.Len(t, res.Postings, 1)
	assert.Equal(t, "JE-2026-0001", res.Postings[0].EntryNumber)
	assert.Equal(t, "2026-01-15", res.Postings[0].EntryDate)
	assert.Equal(t, "USD", res.Postings[0].Currency)
	assert.True(t, res.Postings[0].Amount.Equal(decimal.NewFromInt(1000)))
}

func TestFinanceDraftContext_SearchTransactions(t *testing.T) {
	ctx := context.Background()

	txs := &fakeDraftTransactionService{
		searchRes: []findomain.Transaction{
			{
				ID:              1,
				Number:          "TXN-001",
				TransactionDate: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
				Description:     "Supplier settlement",
				Reference:       "INV-100",
				Currency:        "IDR",
				ExchangeRate:    decimal.NewFromInt(1),
				Source:          "MANUAL",
				Entries:         []findomain.Entry{{ID: 10}, {ID: 11}},
			},
		},
		searchCursor: "cursor-123",
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, txs, nil, nil, nil, nil)
	res, err := adapter.SearchTransactions(ctx, agentsvc.TransactionSearchQuery{
		Reference: "INV-100",
	})
	require.NoError(t, err)
	assert.Len(t, res.Items, 1)
	assert.Equal(t, "TXN-001", res.Items[0].Number)
	assert.Equal(t, "2026-01-20", res.Items[0].TransactionDate)
	assert.Equal(t, 2, res.Items[0].EntryCount)
	assert.Equal(t, "cursor-123", res.NextCursor)
}

func TestFinanceDraftContext_LookupExchangeRate(t *testing.T) {
	ctx := context.Background()

	xr := &fakeDraftExchangeRateService{
		lookupRate: decimal.NewFromInt(16250),
	}

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil, nil, nil, nil, xr)
	res, err := adapter.LookupExchangeRate(ctx, agentsvc.ExchangeRateQuery{
		FromCurrency: "USD",
		ToCurrency:   "IDR",
		Date:         "2026-01-20",
	})
	require.NoError(t, err)
	assert.Equal(t, "USD", res.FromCurrency)
	assert.Equal(t, "IDR", res.ToCurrency)
	assert.True(t, res.Rate.Equal(decimal.NewFromInt(16250)))
}

