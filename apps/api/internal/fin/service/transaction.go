// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
)

// TransactionRepository defines persistence operations for transactions.
type TransactionRepository interface {
	GetByID(c context.Context, id int64) (domain.Transaction, error)
	Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error)
	Update(c context.Context, id int64, req model.UpdateTransactionRequest) (domain.Transaction, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error)
}

// TransactionService manages business logic and validation for financial transactions.
type TransactionService struct {
	repo         TransactionRepository
	periodRepo   PeriodRepository
	accountRepo  AccountRepository
	rateRepo     ExchangeRateRepository
	currencyRepo CurrencyRepository
}

// NewTransactionService constructs a new TransactionService backed by the given repositories.
func NewTransactionService(
	repo TransactionRepository,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
	rateRepo ExchangeRateRepository,
	currencyRepo CurrencyRepository,
) TransactionService {
	return TransactionService{
		repo:         repo,
		periodRepo:   periodRepo,
		accountRepo:  accountRepo,
		rateRepo:     rateRepo,
		currencyRepo: currencyRepo,
	}
}

// GetByID retrieves a transaction by ID.
func (s TransactionService) GetByID(c context.Context, id int64) (domain.Transaction, error) {
	return s.repo.GetByID(c, id)
}

func (s TransactionService) resolveRate(c context.Context, currency string, explicitRate *float64, date time.Time) (float64, error) {
	if explicitRate != nil && *explicitRate > 0 {
		return *explicitRate, nil
	}
	baseCurrency := "IDR"
	if s.currencyRepo != nil {
		wsID := base.WorkspaceIDFromContext(c)
		if wsID > 0 {
			if bc, err := s.currencyRepo.GetBaseCurrency(c, wsID); err == nil && bc != "" {
				baseCurrency = bc
			}
		}
	}
	if currency == "" || currency == baseCurrency {
		return 1.0, nil
	}
	if s.rateRepo == nil {
		return 1.0, nil
	}
	return s.rateRepo.Lookup(c, currency, baseCurrency, date)
}

// Create validates and creates a new transaction.
// If req carries an inline entry, entry validation is executed prior to transaction creation.
func (s TransactionService) Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error) {
	curr := "IDR"
	if req.Currency != "" {
		curr = req.Currency
	}

	rate, err := s.resolveRate(c, curr, req.ExchangeRate, req.TransactionDate.Time())
	if err != nil {
		return domain.Transaction{}, err
	}
	req.ExchangeRate = &rate

	if req.Entry != nil {
		if err := validateEntry(
			c,
			s.periodRepo,
			s.accountRepo,
			req.Entry.PeriodID,
			req.Entry.EntryDate.Time(),
			req.Entry.Postings,
			curr,
			rate,
		); err != nil {
			return domain.Transaction{}, err
		}
	}

	return s.repo.Create(c, req)
}

// Update updates transaction header fields (description, reference).
func (s TransactionService) Update(c context.Context, id int64, req model.UpdateTransactionRequest) (domain.Transaction, error) {
	return s.repo.Update(c, id, req)
}

// Delete soft-deletes a transaction if it has no posted entries.
func (s TransactionService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

// Search searches active transactions matching filter criteria.
func (s TransactionService) Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error) {
	return s.repo.Search(c, req)
}
