package config

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
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
	lastReq *finmodel.CreateTransactionRequest
	tx      findomain.Transaction
	err     error
}

func (f *fakeDraftTransactionService) Create(c context.Context, req finmodel.CreateTransactionRequest) (findomain.Transaction, error) {
	f.lastReq = &req
	if f.err != nil {
		return findomain.Transaction{}, f.err
	}
	return f.tx, nil
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

	adapter := newFinanceDraftContextFromServices(accounts, nil, nil, nil, nil)
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

	adapter := newFinanceDraftContextFromServices(nil, currencies, nil, nil, nil)
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

	adapter := newFinanceDraftContextFromServices(nil, nil, periods, nil, nil)
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

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, cashFlows, nil)

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

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, cashFlows, nil)

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

	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, txs)

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
	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil)

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
	adapterWithCF := newFinanceDraftContextFromServices(nil, nil, nil, cashFlows, nil)

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
	adapterWithTx := newFinanceDraftContextFromServices(nil, nil, nil, nil, txs)
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
	adapter := newFinanceDraftContextFromServices(nil, nil, nil, nil, nil)

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
