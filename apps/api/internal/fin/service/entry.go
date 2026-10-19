// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/money"
)

// EntryRepository defines persistence operations for journal entries and postings.
type EntryRepository interface {
	GetByID(c context.Context, id int64) (domain.Entry, error)
	Create(c context.Context, txID int64, req model.CreateEntryRequest) (domain.Entry, error)
	Update(c context.Context, id int64, req model.UpdateEntryRequest) (domain.Entry, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchEntryRequest) ([]domain.Entry, string, error)
	Post(c context.Context, id int64, postedBy int64) (domain.Entry, error)
	Reverse(c context.Context, id int64, postedBy int64, entryDate time.Time, description string) (domain.Entry, error)
	Void(c context.Context, id int64) (domain.Entry, error)
	GetAccountLedger(c context.Context, accountID int64, page int, limit int) ([]domain.Posting, int64, error)
	GetAccountBalanceBefore(c context.Context, accountID int64, beforePostingID int64) (float64, float64, float64, float64, error)
	GetAccountBalancesAsOf(c context.Context, asOfDate time.Time) (map[int64]repository.AccountBalanceSummary, error)
	HasPostedEntries(c context.Context, workspaceID int64) (bool, error)
	CreateSystemPostedEntry(
		c context.Context,
		periodID int64,
		entryDate time.Time,
		trxDescription string,
		entryDescription string,
		reference string,
		postings []model.CreatePostingRequest,
		postedBy int64,
	) (domain.Entry, error)
}

// EntryService manages business logic and double-entry validation for Journal Entries.
type EntryService struct {
	entryRepo    EntryRepository
	txRepo       TransactionRepository
	periodRepo   PeriodRepository
	accountRepo  AccountRepository
	rateRepo     ExchangeRateRepository
	currencyRepo CurrencyRepository
}

// NewEntryService constructs a new EntryService backed by the given repositories.
func NewEntryService(
	entryRepo EntryRepository,
	txRepo TransactionRepository,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
	rateRepo ExchangeRateRepository,
	currencyRepo CurrencyRepository,
) EntryService {
	return EntryService{
		entryRepo:    entryRepo,
		txRepo:       txRepo,
		periodRepo:   periodRepo,
		accountRepo:  accountRepo,
		rateRepo:     rateRepo,
		currencyRepo: currencyRepo,
	}
}

// GetByID retrieves a journal entry by ID.
func (s EntryService) GetByID(c context.Context, id int64) (domain.Entry, error) {
	return s.entryRepo.GetByID(c, id)
}

func (s EntryService) getBaseCurrencyAndDecimals(c context.Context) (string, int) {
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

func (s EntryService) resolveRate(c context.Context, tx domain.Transaction) (float64, error) {
	if tx.ExchangeRate > 0 {
		return tx.ExchangeRate, nil
	}
	curr := tx.Currency
	if curr == "" {
		curr = "IDR"
	}
	baseCurrency, _ := s.getBaseCurrencyAndDecimals(c)
	if curr == baseCurrency {
		return 1.0, nil
	}
	if s.rateRepo == nil {
		return 1.0, nil
	}
	return s.rateRepo.Lookup(c, curr, baseCurrency, tx.TransactionDate)
}

func findRealizedFXAccounts(c context.Context, accountRepo AccountRepository) (gain domain.Account, loss domain.Account, err error) {
	gain, err = accountRepo.FindByCode(c, "7210")
	if err != nil || gain.ID == 0 {
		accounts, _, _ := accountRepo.Search(c, model.SearchAccountRequest{})
		for _, a := range accounts {
			if a.Code == "7210" || strings.EqualFold(a.Name, "Realized Foreign Exchange Gain") || strings.EqualFold(a.Name, "Realized FX Gain") {
				gain = a
				break
			}
		}
	}

	loss, err = accountRepo.FindByCode(c, "7310")
	if err != nil || loss.ID == 0 {
		accounts, _, _ := accountRepo.Search(c, model.SearchAccountRequest{})
		for _, a := range accounts {
			if a.Code == "7310" || strings.EqualFold(a.Name, "Realized Foreign Exchange Loss") || strings.EqualFold(a.Name, "Realized FX Loss") {
				loss = a
				break
			}
		}
	}

	if gain.ID == 0 || loss.ID == 0 {
		return gain, loss, fmt.Errorf("realized fx accounts (7210 / 7310) not found in workspace")
	}

	return gain, loss, nil
}

// RealizedFXPosting generates a balancing posting for realized FX gain or loss on foreign currency settlement.
// isGain indicates whether the FX difference is a gain (true) or loss (false).
// diff is the absolute difference in base currency.
func (s EntryService) RealizedFXPosting(c context.Context, isGain bool, diff float64, description string) (model.CreatePostingRequest, error) {
	gainAcc, lossAcc, err := findRealizedFXAccounts(c, s.accountRepo)
	if err != nil {
		return model.CreatePostingRequest{}, err
	}

	_, baseDecimals := s.getBaseCurrencyAndDecimals(c)
	roundedDiff := money.Round(math.Abs(diff), baseDecimals)

	if isGain {
		desc := description
		if desc == "" {
			desc = "Realized Foreign Exchange Gain"
		}
		return model.CreatePostingRequest{
			AccountID:   gainAcc.ID,
			Direction:   domain.PostingDirectionCredit,
			Amount:      0,
			BaseAmount:  &roundedDiff,
			Description: desc,
		}, nil
	}

	desc := description
	if desc == "" {
		desc = "Realized Foreign Exchange Loss"
	}
	return model.CreatePostingRequest{
		AccountID:   lossAcc.ID,
		Direction:   domain.PostingDirectionDebit,
		Amount:      0,
		BaseAmount:  &roundedDiff,
		Description: desc,
	}, nil
}


// Create validates and creates a new draft journal entry within a transaction.
func (s EntryService) Create(c context.Context, txID int64, req model.CreateEntryRequest) (domain.Entry, error) {
	tx, err := s.txRepo.GetByID(c, txID)
	if err != nil {
		return domain.Entry{}, err
	}

	curr := tx.Currency
	if curr == "" {
		curr = "IDR"
	}
	rate, err := s.resolveRate(c, tx)
	if err != nil {
		return domain.Entry{}, err
	}

	_, baseDecimals := s.getBaseCurrencyAndDecimals(c)

	balancedPostings, err := validateEntry(
		c,
		s.periodRepo,
		s.accountRepo,
		req.PeriodID,
		req.EntryDate.Time(),
		req.Postings,
		curr,
		rate,
		baseDecimals,
	)
	if err != nil {
		return domain.Entry{}, err
	}
	req.Postings = balancedPostings

	return s.entryRepo.Create(c, txID, req)
}

// Update validates and replaces postings for a draft journal entry.
func (s EntryService) Update(c context.Context, id int64, req model.UpdateEntryRequest) (domain.Entry, error) {
	curr, err := s.entryRepo.GetByID(c, id)
	if err != nil {
		return domain.Entry{}, err
	}
	if curr.Status != domain.EntryStatusDraft {
		return domain.Entry{}, domain.ErrEntryNotDraft
	}

	tx, err := s.txRepo.GetByID(c, curr.TransactionID)
	if err != nil {
		return domain.Entry{}, err
	}

	txCurr := tx.Currency
	if txCurr == "" {
		txCurr = "IDR"
	}
	rate, err := s.resolveRate(c, tx)
	if err != nil {
		return domain.Entry{}, err
	}

	_, baseDecimals := s.getBaseCurrencyAndDecimals(c)

	balancedPostings, err := validateEntry(
		c,
		s.periodRepo,
		s.accountRepo,
		curr.PeriodID,
		curr.EntryDate,
		req.Postings,
		txCurr,
		rate,
		baseDecimals,
	)
	if err != nil {
		return domain.Entry{}, err
	}
	req.Postings = balancedPostings

	return s.entryRepo.Update(c, id, req)
}

// Delete soft-deletes a draft journal entry.
func (s EntryService) Delete(c context.Context, id int64) error {
	curr, err := s.entryRepo.GetByID(c, id)
	if err != nil {
		return err
	}
	if curr.Status != domain.EntryStatusDraft {
		return domain.ErrEntryNotDraft
	}

	return s.entryRepo.Delete(c, id)
}

// Search searches active journal entries matching the filter criteria.
func (s EntryService) Search(c context.Context, req model.SearchEntryRequest) ([]domain.Entry, string, error) {
	return s.entryRepo.Search(c, req)
}

// Post transitions a journal entry from DRAFT to POSTED after running full double-entry validation.
func (s EntryService) Post(c context.Context, id int64, postedBy int64) (domain.Entry, error) {
	curr, err := s.entryRepo.GetByID(c, id)
	if err != nil {
		return domain.Entry{}, err
	}
	if curr.Status != domain.EntryStatusDraft {
		return domain.Entry{}, domain.ErrEntryNotDraft
	}

	tx, err := s.txRepo.GetByID(c, curr.TransactionID)
	if err != nil {
		return domain.Entry{}, err
	}

	txCurr := tx.Currency
	if txCurr == "" {
		txCurr = "IDR"
	}
	rate, err := s.resolveRate(c, tx)
	if err != nil {
		return domain.Entry{}, err
	}

	_, baseDecimals := s.getBaseCurrencyAndDecimals(c)

	postings := make([]model.CreatePostingRequest, len(curr.Postings))
	for i, p := range curr.Postings {
		baseAmt := p.BaseAmount
		postings[i] = model.CreatePostingRequest{
			AccountID:   p.AccountID,
			Direction:   p.Direction,
			Amount:      p.Amount,
			BaseAmount:  &baseAmt,
			Description: p.Description,
		}
	}

	balancedPostings, err := validateEntry(
		c,
		s.periodRepo,
		s.accountRepo,
		curr.PeriodID,
		curr.EntryDate,
		postings,
		txCurr,
		rate,
		baseDecimals,
	)
	if err != nil {
		return domain.Entry{}, err
	}

	if len(balancedPostings) > len(postings) {
		// A balancing posting was added, save to draft entry first
		curr, err = s.entryRepo.Update(c, id, model.UpdateEntryRequest{
			Postings: balancedPostings,
		})
		if err != nil {
			return domain.Entry{}, err
		}
	}

	return s.entryRepo.Post(c, id, postedBy)
}

// Reverse creates a reversing journal entry with POSTED status, swapping DEBIT and CREDIT.
func (s EntryService) Reverse(c context.Context, id int64, postedBy int64, entryDate time.Time, description string) (domain.Entry, error) {
	if entryDate.IsZero() {
		entryDate = time.Now()
	}
	return s.entryRepo.Reverse(c, id, postedBy, entryDate, description)
}

// Void voids a journal entry. If the entry is POSTED, creates a reversing entry and marks the original VOID.
func (s EntryService) Void(c context.Context, id int64, postedBy int64) (domain.Entry, error) {
	curr, err := s.entryRepo.GetByID(c, id)
	if err != nil {
		return domain.Entry{}, err
	}

	if curr.Status == domain.EntryStatusPosted {
		revDate := time.Now()
		desc := "Void reversal of " + curr.Number
		_, err := s.entryRepo.Reverse(c, id, postedBy, revDate, desc)
		if err != nil {
			return domain.Entry{}, err
		}
	}

	return s.entryRepo.Void(c, id)
}

// GetAccountLedger returns postings for an account with running balance, paginated.
func (s EntryService) GetAccountLedger(c context.Context, accountID int64, page int, limit int) ([]domain.Posting, int64, error) {
	acc, err := s.accountRepo.GetByID(c, accountID)
	if err != nil {
		return nil, 0, err
	}

	postings, total, err := s.entryRepo.GetAccountLedger(c, accountID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var startingBalance, startingBaseBalance float64
	if len(postings) > 0 && page > 1 {
		firstID := postings[0].ID
		debits, credits, baseDebits, baseCredits, err := s.entryRepo.GetAccountBalanceBefore(c, accountID, firstID)
		if err == nil {
			if acc.NormalBalance == domain.NormalBalanceDebit {
				startingBalance = debits - credits
				startingBaseBalance = baseDebits - baseCredits
			} else {
				startingBalance = credits - debits
				startingBaseBalance = baseCredits - baseDebits
			}
		}
	}

	running := startingBalance
	runningBase := startingBaseBalance
	for i := range postings {
		amt := postings[i].Amount
		baseAmt := postings[i].BaseAmount
		if acc.NormalBalance == domain.NormalBalanceDebit {
			if postings[i].Direction == domain.PostingDirectionDebit {
				running += amt
				runningBase += baseAmt
			} else {
				running -= amt
				runningBase -= baseAmt
			}
		} else {
			if postings[i].Direction == domain.PostingDirectionCredit {
				running += amt
				runningBase += baseAmt
			} else {
				running -= amt
				runningBase -= baseAmt
			}
		}
		postings[i].RunningBalance = money.Round(running, 4)
		postings[i].RunningBaseBalance = money.Round(runningBase, 4)
		postings[i].Balance = money.Round(running, 4)
		postings[i].BaseBalance = money.Round(runningBase, 4)
	}

	return postings, total, nil
}

// HasPostedEntries reports whether any posted journal entries exist in the given workspace.
func (s EntryService) HasPostedEntries(c context.Context, workspaceID int64) (bool, error) {
	return s.entryRepo.HasPostedEntries(c, workspaceID)
}

// validateEntry validates journal entry postings and rules in strict order:
// 1. Period exists and status OPEN, entry_date within [start_date, end_date]
// 2. Minimal 2 postings
// 3. Each account exists, is_postable == true, is_active == true, amount >= 0
// 4. Account currency matches transaction currency (if account restricts currency)
// 5. sum(debit) == sum(credit) in transaction currency
// 6. sum(base debit) == sum(base credit) in base currency
// validateEntry validates journal entry postings and rules in strict order:
// 1. Period exists and status OPEN, entry_date within [start_date, end_date]
// 2. Minimal 2 postings
// 3. Each account exists, is_postable == true, is_active == true, amount >= 0
// 4. Account currency matches transaction currency (if account restricts currency)
// 5. sum(debit) == sum(credit) in transaction currency
// 6. sum(base debit) == sum(base credit) in base currency (with auto-balancing FX rounding)
func validateEntry(
	ctx context.Context,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
	periodID int64,
	entryDate time.Time,
	postings []model.CreatePostingRequest,
	txCurrency string,
	txExchangeRate float64,
	baseDecimals int,
) ([]model.CreatePostingRequest, error) {
	// 1. Period check
	p, err := periodRepo.GetByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, base.ErrNotFound) || ent.IsNotFound(err) {
			return nil, domain.ErrPeriodNotFound
		}
		return nil, err
	}
	if p.Status == domain.PeriodStatusClosed {
		return nil, domain.ErrPeriodClosed
	}

	d := time.Date(entryDate.Year(), entryDate.Month(), entryDate.Day(), 0, 0, 0, 0, time.UTC)
	start := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(p.EndDate.Year(), p.EndDate.Month(), p.EndDate.Day(), 0, 0, 0, 0, time.UTC)
	if d.Before(start) || d.After(end) {
		return nil, domain.ErrEntryDateOutsidePeriod
	}

	// 2. Minimum 2 postings
	if len(postings) < 2 {
		return nil, domain.ErrEntryMinPostings
	}

	// 3 & 4. Account validity, amount > 0, currency match
	var debitSum, creditSum float64
	var baseDebitSum, baseCreditSum float64

	rate := txExchangeRate
	if rate <= 0 {
		rate = 1.0
	}

	validatedPostings := make([]model.CreatePostingRequest, len(postings))
	copy(validatedPostings, postings)

	for i := range validatedPostings {
		p := &validatedPostings[i]
		if p.Amount < 0 {
			return nil, domain.ErrPostingInvalidAmount
		}
		if p.Amount == 0 && (p.BaseAmount == nil || *p.BaseAmount <= 0) {
			return nil, domain.ErrPostingInvalidAmount
		}

		acc, err := accountRepo.GetByID(ctx, p.AccountID)
		if err != nil {
			return nil, err
		}

		if !acc.IsPostable {
			return nil, domain.ErrPostingAccountNotPostable
		}
		if !acc.IsActive {
			return nil, domain.ErrPostingAccountInactive
		}
		if acc.Currency != "" && acc.Currency != txCurrency {
			return nil, domain.ErrPostingCurrencyMismatch
		}

		baseAmt := money.Round(p.Amount*rate, baseDecimals)
		if p.BaseAmount != nil && *p.BaseAmount > 0 {
			baseAmt = *p.BaseAmount
		} else {
			p.BaseAmount = &baseAmt
		}

		switch p.Direction {
		case domain.PostingDirectionDebit:
			debitSum += p.Amount
			baseDebitSum += baseAmt
		case domain.PostingDirectionCredit:
			creditSum += p.Amount
			baseCreditSum += baseAmt
		default:
			return nil, domain.ErrEntryNotBalanced
		}
	}

	// 5. Balance in transaction currency
	if !money.Equal(debitSum, creditSum, 4) {
		return nil, domain.ErrEntryNotBalanced
	}

	// 6. Balance in base currency (with auto-balancing FX rounding)
	if money.Equal(baseDebitSum, baseCreditSum, baseDecimals) {
		return validatedPostings, nil
	}

	diff := math.Abs(baseDebitSum - baseCreditSum)
	smallestUnit := math.Pow(10, -float64(baseDecimals))

	if diff <= smallestUnit+1e-9 {
		fxAcc, err := findFXRoundingAccount(ctx, accountRepo)
		if err != nil {
			return nil, err
		}

		roundedDiff := money.Round(diff, baseDecimals)
		if roundedDiff == 0 {
			return validatedPostings, nil
		}

		direction := domain.PostingDirectionCredit
		if baseDebitSum < baseCreditSum {
			direction = domain.PostingDirectionDebit
		}

		balancingPosting := model.CreatePostingRequest{
			AccountID:   fxAcc.ID,
			Direction:   direction,
			Amount:      0,
			BaseAmount:  &roundedDiff,
			Description: "Foreign exchange rounding auto-balance",
		}
		balanced := make([]model.CreatePostingRequest, len(validatedPostings), len(validatedPostings)+1)
		copy(balanced, validatedPostings)
		balanced = append(balanced, balancingPosting)
		return balanced, nil
	}

	return nil, domain.ErrEntryBaseNotBalanced
}

func findFXRoundingAccount(c context.Context, accountRepo AccountRepository) (domain.Account, error) {
	// 1. Try finding by code 7400
	acc, err := accountRepo.FindByCode(c, "7400")
	if err == nil && acc.ID > 0 && acc.IsActive && acc.IsPostable {
		return acc, nil
	}

	// 2. Search all accounts in workspace and match by name or code
	accounts, _, err := accountRepo.Search(c, model.SearchAccountRequest{})
	if err == nil {
		for _, a := range accounts {
			if !a.IsActive || !a.IsPostable {
				continue
			}
			if a.Code == "7400" ||
				strings.EqualFold(a.Name, "FX Rounding") ||
				strings.EqualFold(a.Name, "Foreign Exchange Rounding") ||
				strings.EqualFold(a.Name, "Selisih Pembulatan Kurs Valas") {
				return a, nil
			}
		}
		for _, a := range accounts {
			if !a.IsActive || !a.IsPostable {
				continue
			}
			nameLower := strings.ToLower(a.Name)
			if strings.Contains(nameLower, "rounding") || strings.Contains(nameLower, "pembulatan") {
				return a, nil
			}
		}
	}

	return domain.Account{}, domain.ErrEntryBaseNotBalanced
}

