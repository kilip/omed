// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/money"
	"github.com/shopspring/decimal"
)

// PeriodRepository defines the persistence operations required by PeriodService.
type PeriodRepository interface {
	GetByID(c context.Context, id int64) (domain.Period, error)
	FindByDate(c context.Context, date time.Time) (domain.Period, error)
	Search(c context.Context, req model.SearchPeriodRequest) ([]domain.Period, string, error)
	Create(c context.Context, req model.CreatePeriodRequest) (domain.Period, error)
	Update(c context.Context, id int64, req model.UpdatePeriodRequest) (domain.Period, error)
	UpdateStatus(c context.Context, id int64, status string) (domain.Period, error)
	Delete(c context.Context, id int64) error
	HasOverlapping(c context.Context, startDate, endDate time.Time, excludeID int64) (bool, error)
	HasDraftEntries(c context.Context, periodID int64) (bool, error)
	HasRevaluationEntry(c context.Context, periodID int64) (bool, error)
}

// PeriodService manages business logic and validation for Accounting Periods.
type PeriodService struct {
	repo         PeriodRepository
	accountRepo  AccountRepository
	entryRepo    EntryRepository
	rateRepo     ExchangeRateRepository
	currencyRepo CurrencyRepository
	settingRepo  SettingRepository
}

// NewPeriodService constructs a new PeriodService backed by the given repository and optional dependencies.
func NewPeriodService(
	repo PeriodRepository,
	opts ...func(*PeriodService),
) PeriodService {
	s := PeriodService{repo: repo}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// WithPeriodDependencies configures extra dependencies needed for revaluation.
func WithPeriodDependencies(
	accountRepo AccountRepository,
	entryRepo EntryRepository,
	rateRepo ExchangeRateRepository,
	currencyRepo CurrencyRepository,
	settingRepo SettingRepository,
) func(*PeriodService) {
	return func(s *PeriodService) {
		s.accountRepo = accountRepo
		s.entryRepo = entryRepo
		s.rateRepo = rateRepo
		s.currencyRepo = currencyRepo
		s.settingRepo = settingRepo
	}
}

// GetByID retrieves an active period by its ID within a workspace.
func (s PeriodService) GetByID(c context.Context, id int64) (domain.Period, error) {
	return s.repo.GetByID(c, id)
}

// Search searches active periods within a workspace based on filter criteria.
func (s PeriodService) Search(c context.Context, req model.SearchPeriodRequest) ([]domain.Period, string, error) {
	return s.repo.Search(c, req)
}

// Create validates and creates a new accounting period.
func (s PeriodService) Create(c context.Context, req model.CreatePeriodRequest) (domain.Period, error) {
	if req.FiscalYear <= 0 {
		return domain.Period{}, domain.ErrPeriodInvalidFiscalYear
	}
	if req.StartDate.Time().After(req.EndDate.Time()) {
		return domain.Period{}, domain.ErrPeriodInvalidDates
	}

	overlap, err := s.repo.HasOverlapping(c, req.StartDate.Time(), req.EndDate.Time(), 0)
	if err != nil {
		return domain.Period{}, err
	}
	if overlap {
		return domain.Period{}, domain.ErrPeriodOverlap
	}

	return s.repo.Create(c, req)
}

// Update validates and updates an existing accounting period.
func (s PeriodService) Update(c context.Context, id int64, req model.UpdatePeriodRequest) (domain.Period, error) {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if curr.Status == domain.PeriodStatusClosed {
		return domain.Period{}, domain.ErrPeriodClosed
	}

	if req.FiscalYear <= 0 {
		return domain.Period{}, domain.ErrPeriodInvalidFiscalYear
	}
	if req.StartDate.Time().After(req.EndDate.Time()) {
		return domain.Period{}, domain.ErrPeriodInvalidDates
	}

	overlap, err := s.repo.HasOverlapping(c, req.StartDate.Time(), req.EndDate.Time(), id)
	if err != nil {
		return domain.Period{}, err
	}
	if overlap {
		return domain.Period{}, domain.ErrPeriodOverlap
	}

	return s.repo.Update(c, id, req)
}

// Delete validates and soft deletes an existing accounting period.
func (s PeriodService) Delete(c context.Context, id int64) error {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}
	if curr.Status == domain.PeriodStatusClosed {
		return domain.ErrPeriodClosed
	}

	return s.repo.Delete(c, id)
}

// Close locks an open accounting period.
func (s PeriodService) Close(c context.Context, id int64) (domain.Period, error) {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if curr.Status == domain.PeriodStatusClosed {
		return domain.Period{}, domain.ErrPeriodAlreadyClosed
	}

	hasDraft, err := s.repo.HasDraftEntries(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if hasDraft {
		return domain.Period{}, domain.ErrPeriodHasDraftEntries
	}

	return s.repo.UpdateStatus(c, id, domain.PeriodStatusClosed)
}

// Reopen unlocks a closed accounting period.
func (s PeriodService) Reopen(c context.Context, id int64) (domain.Period, error) {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if curr.Status == domain.PeriodStatusOpen {
		return domain.Period{}, domain.ErrPeriodAlreadyOpen
	}

	return s.repo.UpdateStatus(c, id, domain.PeriodStatusOpen)
}

func (s PeriodService) getBaseCurrencyAndDecimals(c context.Context) (string, int) {
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

func findUnrealizedFXAccounts(c context.Context, accountRepo AccountRepository, settingRepo SettingRepository) (gain domain.Account, loss domain.Account, err error) {
	wsID := base.WorkspaceIDFromContext(c)
	if settingRepo == nil || wsID <= 0 {
		return gain, loss, domain.ErrFXAccountsNotConfigured
	}

	setting, err := settingRepo.Get(c, wsID)
	if err != nil || setting.FXUnrealizedGainAccountID == nil || setting.FXUnrealizedLossAccountID == nil {
		return gain, loss, domain.ErrFXAccountsNotConfigured
	}

	gain, err = accountRepo.GetByID(c, *setting.FXUnrealizedGainAccountID)
	if err != nil || gain.ID == 0 {
		return gain, loss, domain.ErrFXGainAccountNotFound
	}
	if !gain.IsActive || !gain.IsPostable {
		return gain, loss, domain.ErrFXAccountInvalid
	}

	loss, err = accountRepo.GetByID(c, *setting.FXUnrealizedLossAccountID)
	if err != nil || loss.ID == 0 {
		return gain, loss, domain.ErrFXLossAccountNotFound
	}
	if !loss.IsActive || !loss.IsPostable {
		return gain, loss, domain.ErrFXAccountInvalid
	}

	return gain, loss, nil
}

// Revalue performs period-end revaluation of foreign currency account balances.
func (s PeriodService) Revalue(ctx context.Context, periodID int64, requestedBy int64) (domain.Entry, error) {
	p, err := s.repo.GetByID(ctx, periodID)
	if err != nil {
		return domain.Entry{}, err
	}
	if p.Status != domain.PeriodStatusOpen {
		return domain.Entry{}, domain.ErrPeriodClosed
	}

	hasRevalued, err := s.repo.HasRevaluationEntry(ctx, periodID)
	if err != nil {
		return domain.Entry{}, err
	}
	if hasRevalued {
		return domain.Entry{}, domain.ErrPeriodAlreadyRevalued
	}

	baseCurrency, baseDecimals := s.getBaseCurrencyAndDecimals(ctx)

	// Fetch all accounts
	accounts, _, err := s.accountRepo.Search(ctx, model.SearchAccountRequest{})
	if err != nil {
		return domain.Entry{}, err
	}

	// Filter accounts with foreign currency
	var foreignAccounts []domain.Account
	for _, a := range accounts {
		if a.Currency != "" && a.Currency != baseCurrency && a.IsActive && a.IsPostable {
			foreignAccounts = append(foreignAccounts, a)
		}
	}

	if len(foreignAccounts) == 0 {
		return domain.Entry{}, nil
	}

	// Fetch cumulative balances as of period.EndDate
	balances, err := s.entryRepo.GetAccountBalancesAsOf(ctx, p.EndDate)
	if err != nil {
		return domain.Entry{}, err
	}

	unrealizedGainAcc, unrealizedLossAcc, err := findUnrealizedFXAccounts(ctx, s.accountRepo, s.settingRepo)
	if err != nil {
		return domain.Entry{}, err
	}

	var postings []model.CreatePostingRequest

	for _, acc := range foreignAccounts {
		bal := balances[acc.ID]

		// Net account balance in account currency
		var netAmount decimal.Decimal
		// Net account balance in base currency currently recorded in ledger
		var currentBaseBalance decimal.Decimal

		if acc.NormalBalance == domain.NormalBalanceDebit {
			netAmount = bal.Debit.Sub(bal.Credit)
			currentBaseBalance = bal.BaseDebit.Sub(bal.BaseCredit)
		} else {
			netAmount = bal.Credit.Sub(bal.Debit)
			currentBaseBalance = bal.BaseCredit.Sub(bal.BaseDebit)
		}

		// If account has zero balance in foreign currency, check if there is an unadjusted base amount
		if money.Equal(netAmount, decimal.Zero, 4) && money.Equal(currentBaseBalance, decimal.Zero, int32(baseDecimals)) {
			continue
		}

		rate, err := s.rateRepo.Lookup(ctx, acc.Currency, baseCurrency, p.EndDate)
		if err != nil {
			return domain.Entry{}, err
		}

		newBaseValue := money.Round(netAmount.Mul(rate), int32(baseDecimals))
		diff := money.Round(newBaseValue.Sub(currentBaseBalance), int32(baseDecimals))

		if money.Equal(diff, decimal.Zero, int32(baseDecimals)) {
			continue
		}

		if diff.IsPositive() {
			// Gain
			absDiff := money.Round(diff, int32(baseDecimals))
			// Posting to foreign account:
			foreignDirection := domain.PostingDirectionDebit
			if acc.NormalBalance == domain.NormalBalanceCredit {
				foreignDirection = domain.PostingDirectionCredit
			}
			postings = append(postings, model.CreatePostingRequest{
				AccountID:   acc.ID,
				Direction:   foreignDirection,
				Amount:      decimal.Zero,
				BaseAmount:  &absDiff,
				Description: fmt.Sprintf("FX Revaluation (%s @ %s)", acc.Currency, rate.StringFixed(4)),
			})
			postings = append(postings, model.CreatePostingRequest{
				AccountID:   unrealizedGainAcc.ID,
				Direction:   domain.PostingDirectionCredit,
				Amount:      decimal.Zero,
				BaseAmount:  &absDiff,
				Description: fmt.Sprintf("Unrealized FX Gain on %s (%s)", acc.Code, acc.Currency),
			})
		} else {
			// Loss (diff < 0)
			absDiff := money.Round(diff.Abs(), int32(baseDecimals))
			// Posting to foreign account:
			foreignDirection := domain.PostingDirectionCredit
			if acc.NormalBalance == domain.NormalBalanceCredit {
				foreignDirection = domain.PostingDirectionDebit
			}
			postings = append(postings, model.CreatePostingRequest{
				AccountID:   acc.ID,
				Direction:   foreignDirection,
				Amount:      decimal.Zero,
				BaseAmount:  &absDiff,
				Description: fmt.Sprintf("FX Revaluation (%s @ %s)", acc.Currency, rate.StringFixed(4)),
			})
			postings = append(postings, model.CreatePostingRequest{
				AccountID:   unrealizedLossAcc.ID,
				Direction:   domain.PostingDirectionDebit,
				Amount:      decimal.Zero,
				BaseAmount:  &absDiff,
				Description: fmt.Sprintf("Unrealized FX Loss on %s (%s)", acc.Code, acc.Currency),
			})
		}
	}

	if len(postings) == 0 {
		return domain.Entry{}, nil
	}

	ref := domain.PeriodRevaluationReference(periodID)
	trxDesc := fmt.Sprintf("Period-end revaluation for %s (%s)", p.Name, p.EndDate.Format("2006-01-02"))
	entryDesc := trxDesc

	return s.entryRepo.CreateSystemPostedEntry(
		ctx,
		p.ID,
		p.EndDate,
		trxDesc,
		entryDesc,
		ref,
		postings,
		requestedBy,
	)
}
