package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/shopspring/decimal"
)

type ExchangeRateRepository interface {
	List(ctx context.Context, req model.ListExchangeRateRequest) ([]model.ExchangeRate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.ExchangeRate, error)
	GetLatestRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (*model.ExchangeRate, error)
	Create(ctx context.Context, req model.CreateExchangeRateRequest) (*model.ExchangeRate, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateExchangeRateRequest) (*model.ExchangeRate, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ExchangeRateService struct {
	repo ExchangeRateRepository
	log  *slog.Logger
}

func NewExchangeRateService(repo ExchangeRateRepository, log *slog.Logger) ExchangeRateService {
	return ExchangeRateService{
		repo: repo,
		log:  log,
	}
}

func (s ExchangeRateService) List(ctx context.Context, req model.ListExchangeRateRequest) ([]model.ExchangeRate, error) {
	return s.repo.List(ctx, req)
}

func (s ExchangeRateService) GetByID(ctx context.Context, id uuid.UUID) (*model.ExchangeRate, error) {
	return s.repo.GetByID(ctx, id)
}

func (s ExchangeRateService) Create(ctx context.Context, req model.CreateExchangeRateRequest) (*model.ExchangeRate, error) {
	if req.FromCurrency == req.ToCurrency {
		return nil, shared.ErrSameCurrencyNotAllowed
	}
	if req.Rate.LessThanOrEqual(decimal.Zero) {
		return nil, shared.ErrInvalidExchangeRate
	}

	return s.repo.Create(ctx, req)
}

func (s ExchangeRateService) Update(ctx context.Context, id uuid.UUID, req model.UpdateExchangeRateRequest) (*model.ExchangeRate, error) {
	if req.Rate != nil && req.Rate.LessThanOrEqual(decimal.Zero) {
		return nil, shared.ErrInvalidExchangeRate
	}

	return s.repo.Update(ctx, id, req)
}

func (s ExchangeRateService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s ExchangeRateService) ConvertToBase(ctx context.Context, amount decimal.Decimal, currency string, baseCurrency string, date time.Time) (decimal.Decimal, decimal.Decimal, error) {
	if currency == baseCurrency {
		return amount, decimal.NewFromInt(1), nil
	}

	rateObj, err := s.repo.GetLatestRate(ctx, currency, baseCurrency, date)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	converted := amount.Mul(rateObj.Rate)
	return converted, rateObj.Rate, nil
}
