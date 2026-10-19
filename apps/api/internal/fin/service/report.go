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
)

// ReportRepository defines persistence queries required for financial reporting.
type ReportRepository interface {
	GetAccountBalancesAsOf(c context.Context, asOfDate time.Time) (map[int64]repository.AccountBalanceSummary, error)
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

	var totalDebit, totalCredit float64
	var accountItems []model.TrialBalanceAccountItem

	for _, acc := range accounts {
		bal := balances[acc.ID]

		// In trial balance: debit and credit are reported in base currency
		baseDebit := money.Round(bal.BaseDebit, baseDecimals)
		baseCredit := money.Round(bal.BaseCredit, baseDecimals)

		// Calculate net balance based on normal balance
		var netBalance float64
		if acc.NormalBalance == domain.NormalBalanceDebit {
			netBalance = money.Round(baseDebit-baseCredit, baseDecimals)
		} else {
			netBalance = money.Round(baseCredit-baseDebit, baseDecimals)
		}

		// Include active or accounts with transaction activity
		if baseDebit == 0 && baseCredit == 0 && !acc.IsActive {
			continue
		}

		totalDebit += baseDebit
		totalCredit += baseCredit

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
				var cNet float64
				if acc.NormalBalance == domain.NormalBalanceDebit {
					cNet = money.Round(cDebit-cCredit, 4)
				} else {
					cNet = money.Round(cCredit-cDebit, 4)
				}

				cbDebit := money.Round(cbal.BaseDebit, baseDecimals)
				cbCredit := money.Round(cbal.BaseCredit, baseDecimals)
				var cbNet float64
				if acc.NormalBalance == domain.NormalBalanceDebit {
					cbNet = money.Round(cbDebit-cbCredit, baseDecimals)
				} else {
					cbNet = money.Round(cbCredit-cbDebit, baseDecimals)
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

	totalDebit = money.Round(totalDebit, baseDecimals)
	totalCredit = money.Round(totalCredit, baseDecimals)
	isBalanced := money.Equal(totalDebit, totalCredit, baseDecimals)

	return model.TrialBalanceResponse{
		AsOfDate:     req.AsOfDate,
		BaseCurrency: baseCurrency,
		Accounts:     accountItems,
		TotalDebit:   totalDebit,
		TotalCredit:  totalCredit,
		IsBalanced:   isBalanced,
	}, nil
}
