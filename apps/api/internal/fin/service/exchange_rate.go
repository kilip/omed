// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/shopspring/decimal"
)

// ExchangeRateRepository defines persistence operations for exchange rates.
type ExchangeRateRepository interface {
	GetByID(c context.Context, id int64) (domain.ExchangeRate, error)
	Create(c context.Context, req model.CreateExchangeRateRequest) (domain.ExchangeRate, error)
	Update(c context.Context, id int64, req model.UpdateExchangeRateRequest) (domain.ExchangeRate, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchExchangeRateRequest) ([]domain.ExchangeRate, string, error)
	Lookup(c context.Context, from, to string, date time.Time) (decimal.Decimal, error)
}

// ExchangeRateService manages business logic and rate resolution for foreign exchange.
type ExchangeRateService struct {
	repo         ExchangeRateRepository
	currencyRepo CurrencyRepository
}

// NewExchangeRateService constructs a new ExchangeRateService instance.
func NewExchangeRateService(repo ExchangeRateRepository, currencyRepo CurrencyRepository) ExchangeRateService {
	return ExchangeRateService{
		repo:         repo,
		currencyRepo: currencyRepo,
	}
}

// GetByID retrieves an exchange rate by ID.
func (s ExchangeRateService) GetByID(c context.Context, id int64) (domain.ExchangeRate, error) {
	return s.repo.GetByID(c, id)
}

// Create validates currencies and persists a new exchange rate.
func (s ExchangeRateService) Create(c context.Context, req model.CreateExchangeRateRequest) (domain.ExchangeRate, error) {
	if s.currencyRepo != nil {
		if _, err := s.currencyRepo.GetByCode(c, req.FromCurrency); err != nil {
			return domain.ExchangeRate{}, err
		}
		if _, err := s.currencyRepo.GetByCode(c, req.ToCurrency); err != nil {
			return domain.ExchangeRate{}, err
		}
	}

	return s.repo.Create(c, req)
}

// Update updates an existing exchange rate.
func (s ExchangeRateService) Update(c context.Context, id int64, req model.UpdateExchangeRateRequest) (domain.ExchangeRate, error) {
	return s.repo.Update(c, id, req)
}

// Delete soft-deletes an exchange rate.
func (s ExchangeRateService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

// Search searches exchange rates matching the filter criteria.
func (s ExchangeRateService) Search(c context.Context, req model.SearchExchangeRateRequest) ([]domain.ExchangeRate, string, error) {
	return s.repo.Search(c, req)
}

// Lookup finds the latest exchange rate for a currency pair effective on or before the given date.
func (s ExchangeRateService) Lookup(c context.Context, from, to string, date time.Time) (decimal.Decimal, error) {
	return s.repo.Lookup(c, from, to, date)
}
