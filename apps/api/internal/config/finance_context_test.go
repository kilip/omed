package config

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/base"
	findomain "github.com/kilip/omed/api/internal/fin/domain"
	finmodel "github.com/kilip/omed/api/internal/fin/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAccountSearcher struct {
	accounts []findomain.Account
	err      error
}

func (f *fakeAccountSearcher) Search(ctx context.Context, req finmodel.SearchAccountRequest) ([]findomain.Account, string, error) {
	if f.err != nil {
		return nil, "", f.err
	}
	return f.accounts, "", nil
}

type fakeCurrencyLister struct {
	currencies []findomain.Currency
	err        error
}

func (f *fakeCurrencyLister) ListCurrencies(ctx context.Context) ([]findomain.Currency, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.currencies, nil
}

type fakeSettingGetter struct {
	settings finmodel.FinSettingResponse
	err      error
}

func (f *fakeSettingGetter) GetSettings(ctx context.Context, workspaceID int64) (finmodel.FinSettingResponse, error) {
	if f.err != nil {
		return finmodel.FinSettingResponse{}, f.err
	}
	return f.settings, nil
}

type fakePeriodSearcher struct {
	periods []findomain.Period
	err     error
	lastReq *finmodel.SearchPeriodRequest
}

func (f *fakePeriodSearcher) Search(ctx context.Context, req finmodel.SearchPeriodRequest) ([]findomain.Period, string, error) {
	f.lastReq = &req
	if f.err != nil {
		return nil, "", f.err
	}
	return f.periods, "", nil
}

func sampleAccounts() []findomain.Account {
	return []findomain.Account{
		// ASSET accounts
		{ID: 1, Code: "1001", Name: "Cash on Hand", Type: findomain.AccountTypeAsset, IsActive: true, IsPostable: true},
		{ID: 2, Code: "1002", Name: "Bank Account", Type: findomain.AccountTypeAsset, IsActive: true, IsPostable: true},
		{ID: 3, Code: "1000", Name: "Current Assets", Type: findomain.AccountTypeAsset, IsActive: true, IsPostable: false}, // non-postable header
		{ID: 4, Code: "1003", Name: "Archived Bank", Type: findomain.AccountTypeAsset, IsActive: false, IsPostable: true},  // inactive

		// REVENUE accounts
		{ID: 10, Code: "4001", Name: "Product Sales", Type: findomain.AccountTypeRevenue, IsActive: true, IsPostable: true},
		{ID: 11, Code: "4002", Name: "Service Income", Type: findomain.AccountTypeRevenue, IsActive: true, IsPostable: true},
		{ID: 12, Code: "4000", Name: "Total Revenue", Type: findomain.AccountTypeRevenue, IsActive: true, IsPostable: false}, // non-postable header
		{ID: 13, Code: "4003", Name: "Old Revenue", Type: findomain.AccountTypeRevenue, IsActive: false, IsPostable: true},   // inactive

		// EXPENSE accounts
		{ID: 20, Code: "5001", Name: "Office Supplies", Type: findomain.AccountTypeExpense, IsActive: true, IsPostable: true},
		{ID: 21, Code: "5002", Name: "Travel Expense", Type: findomain.AccountTypeExpense, IsActive: true, IsPostable: true},
		{ID: 22, Code: "5000", Name: "Operating Expenses", Type: findomain.AccountTypeExpense, IsActive: true, IsPostable: false}, // non-postable header
		{ID: 23, Code: "5003", Name: "Old Expense", Type: findomain.AccountTypeExpense, IsActive: false, IsPostable: true},        // inactive

		// LIABILITY accounts (should not be in payment or category)
		{ID: 30, Code: "2001", Name: "Accounts Payable", Type: findomain.AccountTypeLiability, IsActive: true, IsPostable: true},
	}
}

func sampleCurrencies() []findomain.Currency {
	return []findomain.Currency{
		{Code: "IDR", Name: "Indonesian Rupiah"},
		{Code: "USD", Name: "US Dollar"},
		{Code: "EUR", Name: "Euro"},
	}
}

func TestFinanceContext_Snapshot_Income(t *testing.T) {
	accs := &fakeAccountSearcher{accounts: sampleAccounts()}
	currs := &fakeCurrencyLister{currencies: sampleCurrencies()}
	settings := &fakeSettingGetter{settings: finmodel.FinSettingResponse{BaseCurrency: "USD"}}
	periods := &fakePeriodSearcher{}

	ctxAdapter := newFinanceContextFromServices(accs, currs, settings, periods)

	ctx := base.WithWorkspaceID(context.Background(), 1)
	snap, err := ctxAdapter.Snapshot(ctx, "INCOME")
	require.NoError(t, err)

	// Payment accounts: only active, postable ASSET accounts (IDs 1, 2)
	require.Len(t, snap.PaymentAccounts, 2)
	require.Equal(t, int64(1), snap.PaymentAccounts[0].ID)
	require.Equal(t, "1001", snap.PaymentAccounts[0].Code)
	require.Equal(t, "Cash on Hand", snap.PaymentAccounts[0].Name)
	require.Equal(t, int64(2), snap.PaymentAccounts[1].ID)

	// Category accounts: only active, postable REVENUE accounts (IDs 10, 11)
	require.Len(t, snap.CategoryAccounts, 2)
	require.Equal(t, int64(10), snap.CategoryAccounts[0].ID)
	require.Equal(t, "4001", snap.CategoryAccounts[0].Code)
	require.Equal(t, "Product Sales", snap.CategoryAccounts[0].Name)
	require.Equal(t, int64(11), snap.CategoryAccounts[1].ID)

	// Currencies: reference codes
	require.Equal(t, []string{"IDR", "USD", "EUR"}, snap.Currencies)

	// Base currency
	require.Equal(t, "USD", snap.BaseCurrency)
	require.True(t, snap.HasPeriodContext)
	require.Empty(t, snap.OpenPeriods)
}

func TestFinanceContext_Snapshot_Expense(t *testing.T) {
	accs := &fakeAccountSearcher{accounts: sampleAccounts()}
	currs := &fakeCurrencyLister{currencies: sampleCurrencies()}
	settings := &fakeSettingGetter{settings: finmodel.FinSettingResponse{BaseCurrency: "EUR"}}
	periods := &fakePeriodSearcher{}

	ctxAdapter := newFinanceContextFromServices(accs, currs, settings, periods)

	ctx := base.WithWorkspaceID(context.Background(), 1)
	snap, err := ctxAdapter.Snapshot(ctx, "EXPENSE")
	require.NoError(t, err)

	// Payment accounts: only active, postable ASSET accounts
	require.Len(t, snap.PaymentAccounts, 2)
	require.Equal(t, int64(1), snap.PaymentAccounts[0].ID)
	require.Equal(t, int64(2), snap.PaymentAccounts[1].ID)

	// Category accounts: only active, postable EXPENSE accounts (IDs 20, 21)
	require.Len(t, snap.CategoryAccounts, 2)
	require.Equal(t, int64(20), snap.CategoryAccounts[0].ID)
	require.Equal(t, "5001", snap.CategoryAccounts[0].Code)
	require.Equal(t, "Office Supplies", snap.CategoryAccounts[0].Name)
	require.Equal(t, int64(21), snap.CategoryAccounts[1].ID)

	// Currencies
	require.Equal(t, []string{"IDR", "USD", "EUR"}, snap.Currencies)

	// Base currency
	require.Equal(t, "EUR", snap.BaseCurrency)
	require.True(t, snap.HasPeriodContext)
	require.Empty(t, snap.OpenPeriods)
}

func TestFinanceContext_Snapshot_DefaultBaseCurrency(t *testing.T) {
	accs := &fakeAccountSearcher{accounts: sampleAccounts()}
	currs := &fakeCurrencyLister{currencies: sampleCurrencies()}
	settings := &fakeSettingGetter{settings: finmodel.FinSettingResponse{BaseCurrency: ""}}
	periods := &fakePeriodSearcher{}

	ctxAdapter := newFinanceContextFromServices(accs, currs, settings, periods)

	ctx := base.WithWorkspaceID(context.Background(), 1)
	snap, err := ctxAdapter.Snapshot(ctx, "INCOME")
	require.NoError(t, err)

	// Default fallback to "IDR"
	require.Equal(t, "IDR", snap.BaseCurrency)
}

func TestFinanceContext_Snapshot_OpenPeriods(t *testing.T) {
	accs := &fakeAccountSearcher{accounts: sampleAccounts()}
	currs := &fakeCurrencyLister{currencies: sampleCurrencies()}
	settings := &fakeSettingGetter{settings: finmodel.FinSettingResponse{BaseCurrency: "IDR"}}
	periods := &fakePeriodSearcher{
		periods: []findomain.Period{
			{
				ID:        1,
				Name:      "Jan 2026",
				Status:    findomain.PeriodStatusOpen,
				StartDate: time.Date(2026, 1, 1, 15, 30, 0, 0, time.FixedZone("WIB", 7*3600)),
				EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.FixedZone("WIB", 7*3600)),
			},
			{
				ID:        2,
				Name:      "Dec 2025",
				Status:    findomain.PeriodStatusClosed,
				StartDate: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
			},
		},
	}

	ctxAdapter := newFinanceContextFromServices(accs, currs, settings, periods)
	ctx := base.WithWorkspaceID(context.Background(), 1)
	snap, err := ctxAdapter.Snapshot(ctx, "EXPENSE")
	require.NoError(t, err)

	require.True(t, snap.HasPeriodContext)
	require.Len(t, snap.OpenPeriods, 1)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), snap.OpenPeriods[0].StartDate)
	assert.Equal(t, time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), snap.OpenPeriods[0].EndDate)

	require.NotNil(t, periods.lastReq)
	require.NotNil(t, periods.lastReq.Status)
	assert.Equal(t, findomain.PeriodStatusOpen, *periods.lastReq.Status)
}

func TestFinanceContext_Snapshot_PeriodSearchError(t *testing.T) {
	accs := &fakeAccountSearcher{accounts: sampleAccounts()}
	currs := &fakeCurrencyLister{currencies: sampleCurrencies()}
	settings := &fakeSettingGetter{settings: finmodel.FinSettingResponse{BaseCurrency: "IDR"}}
	periods := &fakePeriodSearcher{err: errors.New("database connection failed")}

	ctxAdapter := newFinanceContextFromServices(accs, currs, settings, periods)
	ctx := base.WithWorkspaceID(context.Background(), 1)
	_, err := ctxAdapter.Snapshot(ctx, "EXPENSE")
	require.Error(t, err)
}

func TestFinanceContext_Snapshot_NilPeriods(t *testing.T) {
	accs := &fakeAccountSearcher{accounts: sampleAccounts()}
	currs := &fakeCurrencyLister{currencies: sampleCurrencies()}
	settings := &fakeSettingGetter{settings: finmodel.FinSettingResponse{BaseCurrency: "IDR"}}

	ctxAdapter := newFinanceContextFromServices(accs, currs, settings, nil)
	ctx := base.WithWorkspaceID(context.Background(), 1)
	snap, err := ctxAdapter.Snapshot(ctx, "EXPENSE")
	require.NoError(t, err)
	assert.False(t, snap.HasPeriodContext)
	assert.Nil(t, snap.OpenPeriods)
}

func TestFinanceContext_NilFinModule(t *testing.T) {
	fc := newFinanceContext(nil)
	require.Nil(t, fc)
}
