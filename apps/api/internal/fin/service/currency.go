// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"

	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
)

// CurrencyRepository defines persistence operations for currency and workspace currency settings.
type CurrencyRepository interface {
	List(c context.Context) ([]domain.Currency, error)
	GetByCode(c context.Context, code string) (domain.Currency, error)
	GetBaseCurrency(c context.Context, workspaceID int64) (string, error)
	UpdateWorkspaceBaseCurrency(c context.Context, workspaceID int64, baseCurrency string) (model.WorkspaceBaseCurrencyResponse, error)
}

// CurrencyService manages business logic for reference currencies and workspace base currency configuration.
type CurrencyService struct {
	currencyRepo CurrencyRepository
	entryRepo    EntryRepository
}

// NewCurrencyService constructs a new CurrencyService instance.
func NewCurrencyService(currencyRepo CurrencyRepository, entryRepo EntryRepository) CurrencyService {
	return CurrencyService{
		currencyRepo: currencyRepo,
		entryRepo:    entryRepo,
	}
}

// ListCurrencies returns all supported global currencies.
func (s CurrencyService) ListCurrencies(c context.Context) ([]domain.Currency, error) {
	return s.currencyRepo.List(c)
}

// UpdateBaseCurrency modifies the base currency for a workspace.
// If the workspace already has posted journal entries, the operation is rejected with ErrBaseCurrencyLocked.
func (s CurrencyService) UpdateBaseCurrency(c context.Context, workspaceID int64, req model.UpdateBaseCurrencyRequest) (model.WorkspaceBaseCurrencyResponse, error) {
	// 1. Verify currency exists in global currencies
	if _, err := s.currencyRepo.GetByCode(c, req.BaseCurrency); err != nil {
		return model.WorkspaceBaseCurrencyResponse{}, err
	}

	// 2. Check if workspace has posted entries
	hasPosted, err := s.entryRepo.HasPostedEntries(c, workspaceID)
	if err != nil {
		return model.WorkspaceBaseCurrencyResponse{}, err
	}
	if hasPosted {
		return model.WorkspaceBaseCurrencyResponse{}, domain.ErrBaseCurrencyLocked
	}

	// 3. Update workspace base currency
	return s.currencyRepo.UpdateWorkspaceBaseCurrency(c, workspaceID, req.BaseCurrency)
}
