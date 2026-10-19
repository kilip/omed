// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
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
	GetAccountBalanceBefore(c context.Context, accountID int64, beforePostingID int64) (float64, float64, error)
}

// EntryService manages business logic and double-entry validation for Journal Entries.
type EntryService struct {
	entryRepo   EntryRepository
	txRepo      TransactionRepository
	periodRepo  PeriodRepository
	accountRepo AccountRepository
}

// NewEntryService constructs a new EntryService backed by the given repositories.
func NewEntryService(
	entryRepo EntryRepository,
	txRepo TransactionRepository,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
) EntryService {
	return EntryService{
		entryRepo:   entryRepo,
		txRepo:      txRepo,
		periodRepo:  periodRepo,
		accountRepo: accountRepo,
	}
}

// GetByID retrieves a journal entry by ID.
func (s EntryService) GetByID(c context.Context, id int64) (domain.Entry, error) {
	return s.entryRepo.GetByID(c, id)
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
	rate := tx.ExchangeRate
	if rate <= 0 {
		rate = 1.0
	}

	if err := validateEntry(
		c,
		s.periodRepo,
		s.accountRepo,
		req.PeriodID,
		req.EntryDate.Time(),
		req.Postings,
		curr,
		rate,
	); err != nil {
		return domain.Entry{}, err
	}

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
	rate := tx.ExchangeRate
	if rate <= 0 {
		rate = 1.0
	}

	if err := validateEntry(
		c,
		s.periodRepo,
		s.accountRepo,
		curr.PeriodID,
		curr.EntryDate,
		req.Postings,
		txCurr,
		rate,
	); err != nil {
		return domain.Entry{}, err
	}

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
	rate := tx.ExchangeRate
	if rate <= 0 {
		rate = 1.0
	}

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

	if err := validateEntry(
		c,
		s.periodRepo,
		s.accountRepo,
		curr.PeriodID,
		curr.EntryDate,
		postings,
		txCurr,
		rate,
	); err != nil {
		return domain.Entry{}, err
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

	var startingBalance float64
	if len(postings) > 0 && page > 1 {
		firstID := postings[0].ID
		debits, credits, err := s.entryRepo.GetAccountBalanceBefore(c, accountID, firstID)
		if err == nil {
			if acc.NormalBalance == domain.NormalBalanceDebit {
				startingBalance = debits - credits
			} else {
				startingBalance = credits - debits
			}
		}
	}

	running := startingBalance
	for i := range postings {
		amt := postings[i].Amount
		if acc.NormalBalance == domain.NormalBalanceDebit {
			if postings[i].Direction == domain.PostingDirectionDebit {
				running += amt
			} else {
				running -= amt
			}
		} else {
			if postings[i].Direction == domain.PostingDirectionCredit {
				running += amt
			} else {
				running -= amt
			}
		}
		postings[i].RunningBalance = running
	}

	return postings, total, nil
}

// validateEntry validates journal entry postings and rules in strict order:
// 1. Period exists and status OPEN, entry_date within [start_date, end_date]
// 2. Minimal 2 postings
// 3. Each account exists, is_postable == true, is_active == true, amount > 0
// 4. Account currency matches transaction currency (if account restricts currency)
// 5. sum(debit) == sum(credit) in transaction currency
// 6. sum(base debit) == sum(base credit) in base currency
func validateEntry(
	ctx context.Context,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
	periodID int64,
	entryDate time.Time,
	postings []model.CreatePostingRequest,
	txCurrency string,
	txExchangeRate float64,
) error {
	// 1. Period check
	p, err := periodRepo.GetByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, base.ErrNotFound) || ent.IsNotFound(err) {
			return domain.ErrPeriodNotFound
		}
		return err
	}
	if p.Status == domain.PeriodStatusClosed {
		return domain.ErrPeriodClosed
	}

	d := time.Date(entryDate.Year(), entryDate.Month(), entryDate.Day(), 0, 0, 0, 0, time.UTC)
	start := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(p.EndDate.Year(), p.EndDate.Month(), p.EndDate.Day(), 0, 0, 0, 0, time.UTC)
	if d.Before(start) || d.After(end) {
		return domain.ErrEntryDateOutsidePeriod
	}

	// 2. Minimum 2 postings
	if len(postings) < 2 {
		return domain.ErrEntryMinPostings
	}

	// 3 & 4. Account validity, amount > 0, currency match
	var debitSum, creditSum float64
	var baseDebitSum, baseCreditSum float64

	rate := txExchangeRate
	if rate <= 0 {
		rate = 1.0
	}

	for _, p := range postings {
		if p.Amount <= 0 {
			return domain.ErrPostingInvalidAmount
		}

		acc, err := accountRepo.GetByID(ctx, p.AccountID)
		if err != nil {
			return err
		}

		if !acc.IsPostable {
			return domain.ErrPostingAccountNotPostable
		}
		if !acc.IsActive {
			return domain.ErrPostingAccountInactive
		}
		if acc.Currency != "" && acc.Currency != txCurrency {
			return domain.ErrPostingCurrencyMismatch
		}

		baseAmt := p.Amount * rate
		if p.BaseAmount != nil && *p.BaseAmount > 0 {
			baseAmt = *p.BaseAmount
		}

		switch p.Direction {
		case domain.PostingDirectionDebit:
			debitSum += p.Amount
			baseDebitSum += baseAmt
		case domain.PostingDirectionCredit:
			creditSum += p.Amount
			baseCreditSum += baseAmt
		default:
			return domain.ErrEntryNotBalanced
		}
	}

	// 5. Balance in transaction currency
	if math.Abs(debitSum-creditSum) > 1e-4 {
		return domain.ErrEntryNotBalanced
	}

	// 6. Balance in base currency
	if math.Abs(baseDebitSum-baseCreditSum) > 1e-4 {
		return domain.ErrEntryBaseNotBalanced
	}

	return nil
}
