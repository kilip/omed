// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"sort"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/money"
	"github.com/shopspring/decimal"
)

// ReportRepository defines persistence queries required for financial reporting.
type ReportRepository interface {
	GetAccountBalancesAsOf(c context.Context, asOfDate time.Time) (map[int64]repository.AccountBalanceSummary, error)
	GetMonthlyAccountFlows(c context.Context, startDate time.Time, endDate time.Time) ([]repository.MonthlyAccountFlow, error)
}

// ReportService coordinates generation of financial statements and reports.
type ReportService struct {
	entryRepo    ReportRepository
	accountRepo  AccountRepository
	currencyRepo CurrencyRepository
}

// NewReportService constructs a new ReportService.
func NewReportService(
	entryRepo ReportRepository,
	accountRepo AccountRepository,
	currencyRepo CurrencyRepository,
) ReportService {
	return ReportService{
		entryRepo:    entryRepo,
		accountRepo:  accountRepo,
		currencyRepo: currencyRepo,
	}
}

func (s ReportService) getBaseCurrencyAndDecimals(c context.Context) (string, int) {
	baseCurrency := "IDR"
	baseDecimals := 0
	wsID := base.WorkspaceIDFromContext(c)
	if s.currencyRepo != nil && wsID > 0 {
		if bc, err := s.currencyRepo.GetBaseCurrency(c, wsID); err == nil && bc != "" {
			baseCurrency = bc
		}
	}
	if s.currencyRepo != nil {
		if cur, err := s.currencyRepo.GetByCode(c, baseCurrency); err == nil {
			baseDecimals = cur.Decimals
		} else if baseCurrency == "USD" {
			baseDecimals = 2
		}
	}
	return baseCurrency, baseDecimals
}

// BaseCurrency returns the base currency configured for the workspace in context.
func (s ReportService) BaseCurrency(ctx context.Context) string {
	baseCurrency, _ := s.getBaseCurrencyAndDecimals(ctx)
	return baseCurrency
}

// TrialBalance generates the Trial Balance report as of a specified date.
func (s ReportService) TrialBalance(ctx context.Context, req model.TrialBalanceRequest) (model.TrialBalanceResponse, error) {
	asOf := req.AsOfDate.Time()
	baseCurrency, baseDecimals := s.getBaseCurrencyAndDecimals(ctx)

	// Fetch all accounts in workspace
	accounts, _, err := s.accountRepo.Search(ctx, model.SearchAccountRequest{})
	if err != nil {
		return model.TrialBalanceResponse{}, err
	}

	// Fetch posted balances as of the date
	balances, err := s.entryRepo.GetAccountBalancesAsOf(ctx, asOf)
	if err != nil {
		return model.TrialBalanceResponse{}, err
	}

	includeBreakdown := req.IncludePerCurrencyBreakdown != nil && *req.IncludePerCurrencyBreakdown

	var totalDebit, totalCredit decimal.Decimal
	var accountItems []model.TrialBalanceAccountItem

	for _, acc := range accounts {
		bal := balances[acc.ID]

		// In trial balance: debit and credit are reported in base currency
		baseDebit := money.Round(bal.BaseDebit, int32(baseDecimals))
		baseCredit := money.Round(bal.BaseCredit, int32(baseDecimals))

		// Calculate net balance based on normal balance
		var netBalance decimal.Decimal
		if acc.NormalBalance == domain.NormalBalanceDebit {
			netBalance = money.Round(baseDebit.Sub(baseCredit), int32(baseDecimals))
		} else {
			netBalance = money.Round(baseCredit.Sub(baseDebit), int32(baseDecimals))
		}

		// Include active or accounts with transaction activity
		if baseDebit.IsZero() && baseCredit.IsZero() && !acc.IsActive {
			continue
		}

		totalDebit = totalDebit.Add(baseDebit)
		totalCredit = totalCredit.Add(baseCredit)

		item := model.TrialBalanceAccountItem{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			Type:        acc.Type,
			Debit:       baseDebit,
			Credit:      baseCredit,
			NetBalance:  netBalance,
		}

		if includeBreakdown && len(bal.Currencies) > 0 {
			var currencies []model.CurrencyBreakdownItem
			for currCode, cbal := range bal.Currencies {
				cDebit := money.Round(cbal.Debit, 4)
				cCredit := money.Round(cbal.Credit, 4)
				var cNet decimal.Decimal
				if acc.NormalBalance == domain.NormalBalanceDebit {
					cNet = money.Round(cDebit.Sub(cCredit), 4)
				} else {
					cNet = money.Round(cCredit.Sub(cDebit), 4)
				}

				cbDebit := money.Round(cbal.BaseDebit, int32(baseDecimals))
				cbCredit := money.Round(cbal.BaseCredit, int32(baseDecimals))
				var cbNet decimal.Decimal
				if acc.NormalBalance == domain.NormalBalanceDebit {
					cbNet = money.Round(cbDebit.Sub(cbCredit), int32(baseDecimals))
				} else {
					cbNet = money.Round(cbCredit.Sub(cbDebit), int32(baseDecimals))
				}

				currencies = append(currencies, model.CurrencyBreakdownItem{
					Currency:   currCode,
					Debit:      cDebit,
					Credit:     cCredit,
					NetBalance: cNet,
					BaseDebit:  cbDebit,
					BaseCredit: cbCredit,
					BaseNet:    cbNet,
				})
			}
			sort.Slice(currencies, func(i, j int) bool {
				return currencies[i].Currency < currencies[j].Currency
			})
			item.Currencies = currencies
		}

		accountItems = append(accountItems, item)
	}

	totalDebit = money.Round(totalDebit, int32(baseDecimals))
	totalCredit = money.Round(totalCredit, int32(baseDecimals))
	isBalanced := money.Equal(totalDebit, totalCredit, int32(baseDecimals))

	return model.TrialBalanceResponse{
		AsOfDate:     req.AsOfDate,
		BaseCurrency: baseCurrency,
		Accounts:     accountItems,
		TotalDebit:   totalDebit,
		TotalCredit:  totalCredit,
		IsBalanced:   isBalanced,
	}, nil
}

// Summary calculates cumulative financial totals and monthly revenue vs expense series.
func (s ReportService) Summary(ctx context.Context, months int) (model.FinancialSummaryResponse, error) {
	if months <= 0 || months > 24 {
		months = 6
	}

	baseCurrency, baseDecimals := s.getBaseCurrencyAndDecimals(ctx)

	// Fetch all accounts in workspace
	accounts, _, err := s.accountRepo.Search(ctx, model.SearchAccountRequest{})
	if err != nil {
		return model.FinancialSummaryResponse{}, err
	}

	now := time.Now()
	balances, err := s.entryRepo.GetAccountBalancesAsOf(ctx, now)
	if err != nil {
		return model.FinancialSummaryResponse{}, err
	}

	var totals model.FinancialSummaryTotals
	for _, acc := range accounts {
		bal := balances[acc.ID]
		baseDebit := money.Round(bal.BaseDebit, int32(baseDecimals))
		baseCredit := money.Round(bal.BaseCredit, int32(baseDecimals))

		switch acc.Type {
		case domain.AccountTypeAsset:
			totals.Assets = totals.Assets.Add(baseDebit.Sub(baseCredit))
		case domain.AccountTypeLiability:
			totals.Liabilities = totals.Liabilities.Add(baseCredit.Sub(baseDebit))
		case domain.AccountTypeEquity:
			totals.Equity = totals.Equity.Add(baseCredit.Sub(baseDebit))
		case domain.AccountTypeRevenue:
			totals.Revenue = totals.Revenue.Add(baseCredit.Sub(baseDebit))
		case domain.AccountTypeExpense:
			totals.Expense = totals.Expense.Add(baseDebit.Sub(baseCredit))
		}
	}

	totals.Assets = money.Round(totals.Assets, int32(baseDecimals))
	totals.Liabilities = money.Round(totals.Liabilities, int32(baseDecimals))
	totals.Equity = money.Round(totals.Equity, int32(baseDecimals))
	totals.Revenue = money.Round(totals.Revenue, int32(baseDecimals))
	totals.Expense = money.Round(totals.Expense, int32(baseDecimals))
	totals.NetIncome = money.Round(totals.Revenue.Sub(totals.Expense), int32(baseDecimals))

	year, month, _ := now.Date()
	startMonthTime := time.Date(year, month, 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
	monthlyItems, err := s.MonthlyRevenueExpense(ctx, startMonthTime, months)
	if err != nil {
		return model.FinancialSummaryResponse{}, err
	}

	return model.FinancialSummaryResponse{
		BaseCurrency: baseCurrency,
		Totals:       totals,
		Monthly:      monthlyItems,
	}, nil
}

// MonthlyRevenueExpense calculates monthly revenue and expense series starting from firstMonth for the given number of months.
func (s ReportService) MonthlyRevenueExpense(ctx context.Context, firstMonth time.Time, months int) ([]model.MonthlyRevenueExpenseItem, error) {
	if months <= 0 {
		return []model.MonthlyRevenueExpenseItem{}, nil
	}

	_, baseDecimals := s.getBaseCurrencyAndDecimals(ctx)

	// Fetch all accounts in workspace
	accounts, _, err := s.accountRepo.Search(ctx, model.SearchAccountRequest{})
	if err != nil {
		return nil, err
	}

	accountMap := make(map[int64]domain.Account, len(accounts))
	for _, acc := range accounts {
		accountMap[acc.ID] = acc
	}

	// Build list of months starting from firstMonth
	monthList := make([]string, months)
	year, month, _ := firstMonth.Date()
	startMonthTime := time.Date(year, month, 1, 0, 0, 0, 0, firstMonth.Location())
	for i := 0; i < months; i++ {
		m := startMonthTime.AddDate(0, i, 0)
		monthList[i] = m.Format("2006-01")
	}

	startDate := startMonthTime
	endOfMonth := time.Date(year, month+time.Month(months), 0, 23, 59, 59, 999999999, firstMonth.Location())
	flows, err := s.entryRepo.GetMonthlyAccountFlows(ctx, startDate, endOfMonth)
	if err != nil {
		return nil, err
	}

	type monthlyAccSum struct {
		revenue decimal.Decimal
		expense decimal.Decimal
	}
	monthlySums := make(map[string]*monthlyAccSum)
	for _, m := range monthList {
		monthlySums[m] = &monthlyAccSum{}
	}

	for _, f := range flows {
		acc, ok := accountMap[f.AccountID]
		if !ok {
			continue
		}
		ms, ok := monthlySums[f.Month]
		if !ok {
			continue
		}
		switch acc.Type {
		case domain.AccountTypeRevenue:
			ms.revenue = ms.revenue.Add(f.BaseCredit.Sub(f.BaseDebit))
		case domain.AccountTypeExpense:
			ms.expense = ms.expense.Add(f.BaseDebit.Sub(f.BaseCredit))
		}
	}

	monthlyItems := make([]model.MonthlyRevenueExpenseItem, 0, months)
	for _, m := range monthList {
		ms := monthlySums[m]
		monthlyItems = append(monthlyItems, model.MonthlyRevenueExpenseItem{
			Month:   m,
			Revenue: money.Round(ms.revenue, int32(baseDecimals)),
			Expense: money.Round(ms.expense, int32(baseDecimals)),
		})
	}

	return monthlyItems, nil
}
