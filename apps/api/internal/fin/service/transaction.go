// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"

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
	repo        TransactionRepository
	periodRepo  PeriodRepository
	accountRepo AccountRepository
}

// NewTransactionService constructs a new TransactionService backed by the given repositories.
func NewTransactionService(
	repo TransactionRepository,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
) TransactionService {
	return TransactionService{
		repo:        repo,
		periodRepo:  periodRepo,
		accountRepo: accountRepo,
	}
}

// GetByID retrieves a transaction by ID.
func (s TransactionService) GetByID(c context.Context, id int64) (domain.Transaction, error) {
	return s.repo.GetByID(c, id)
}

// Create validates and creates a new transaction.
// If req carries an inline entry, entry validation is executed prior to transaction creation.
func (s TransactionService) Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error) {
	if req.Entry != nil {
		curr := "IDR"
		if req.Currency != "" {
			curr = req.Currency
		}

		rate := 1.0
		if req.ExchangeRate != nil && *req.ExchangeRate > 0 {
			rate = *req.ExchangeRate
		}

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

// Delete marks a transaction as deleted.
func (s TransactionService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

// Search searches active transactions matching the filter criteria.
func (s TransactionService) Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error) {
	return s.repo.Search(c, req)
}
