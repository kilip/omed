package config

import (
	"context"
	"strings"
	"time"

	agentsvc "github.com/kilip/omed/api/internal/agent/service"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin"
	findomain "github.com/kilip/omed/api/internal/fin/domain"
	finmodel "github.com/kilip/omed/api/internal/fin/model"
)

type accountSearcher interface {
	Search(c context.Context, req finmodel.SearchAccountRequest) ([]findomain.Account, string, error)
}

type currencyLister interface {
	ListCurrencies(c context.Context) ([]findomain.Currency, error)
}

type settingGetter interface {
	GetSettings(c context.Context, workspaceID int64) (finmodel.FinSettingResponse, error)
}

type periodSearcher interface {
	Search(c context.Context, req finmodel.SearchPeriodRequest) ([]findomain.Period, string, error)
}

type financeContextAdapter struct {
	accounts   accountSearcher
	currencies currencyLister
	settings   settingGetter
	periods    periodSearcher
}

func newFinanceContext(finMod *fin.FinModule) agentsvc.FinanceContext {
	if finMod == nil {
		return nil
	}
	return newFinanceContextFromServices(finMod.AccountService, finMod.CurrencyService, finMod.SettingService, finMod.PeriodService)
}

func newFinanceContextFromServices(accounts accountSearcher, currencies currencyLister, settings settingGetter, periods periodSearcher) agentsvc.FinanceContext {
	return &financeContextAdapter{
		accounts:   accounts,
		currencies: currencies,
		settings:   settings,
		periods:    periods,
	}
}

func (a *financeContextAdapter) Snapshot(ctx context.Context, flow string) (agentsvc.FinanceSnapshot, error) {
	paymentAccounts := make([]agentsvc.AccountOption, 0)
	categoryAccounts := make([]agentsvc.AccountOption, 0)

	targetCategoryType := ""
	switch strings.ToUpper(strings.TrimSpace(flow)) {
	case "INCOME":
		targetCategoryType = findomain.AccountTypeRevenue
	case "EXPENSE":
		targetCategoryType = findomain.AccountTypeExpense
	}

	if a.accounts != nil {
		accs, _, err := a.accounts.Search(ctx, finmodel.SearchAccountRequest{})
		if err != nil {
			return agentsvc.FinanceSnapshot{}, err
		}
		for _, acc := range accs {
			if !acc.IsActive || !acc.IsPostable {
				continue
			}
			if acc.Type == findomain.AccountTypeAsset {
				paymentAccounts = append(paymentAccounts, agentsvc.AccountOption{
					ID:   acc.ID,
					Code: acc.Code,
					Name: acc.Name,
				})
			}
			if targetCategoryType != "" && acc.Type == targetCategoryType {
				categoryAccounts = append(categoryAccounts, agentsvc.AccountOption{
					ID:   acc.ID,
					Code: acc.Code,
					Name: acc.Name,
				})
			}
		}
	}

	curCodes := make([]string, 0)
	if a.currencies != nil {
		currencies, err := a.currencies.ListCurrencies(ctx)
		if err != nil {
			return agentsvc.FinanceSnapshot{}, err
		}
		for _, c := range currencies {
			curCodes = append(curCodes, c.Code)
		}
	}

	baseCurrency := "IDR"
	if a.settings != nil {
		wsID := base.WorkspaceIDFromContext(ctx)
		s, err := a.settings.GetSettings(ctx, wsID)
		if err == nil && s.BaseCurrency != "" {
			baseCurrency = s.BaseCurrency
		}
	}

	var openPeriods []agentsvc.PeriodRange
	hasPeriodContext := false
	if a.periods != nil {
		openStatus := findomain.PeriodStatusOpen
		periods, _, err := a.periods.Search(ctx, finmodel.SearchPeriodRequest{Status: &openStatus})
		if err != nil {
			return agentsvc.FinanceSnapshot{}, err
		}
		openPeriods = make([]agentsvc.PeriodRange, 0)
		for _, p := range periods {
			if p.Status == findomain.PeriodStatusOpen {
				startDate := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
				endDate := time.Date(p.EndDate.Year(), p.EndDate.Month(), p.EndDate.Day(), 0, 0, 0, 0, time.UTC)
				openPeriods = append(openPeriods, agentsvc.PeriodRange{
					StartDate: startDate,
					EndDate:   endDate,
				})
			}
		}
		hasPeriodContext = true
	}

	return agentsvc.FinanceSnapshot{
		PaymentAccounts:  paymentAccounts,
		CategoryAccounts: categoryAccounts,
		Currencies:       curCodes,
		BaseCurrency:     baseCurrency,
		OpenPeriods:      openPeriods,
		HasPeriodContext: hasPeriodContext,
	}, nil
}
