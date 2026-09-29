package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/exchangerate"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type ExchangeRateRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func toExchangeRate(e *ent.ExchangeRate) *model.ExchangeRate {
	if e == nil {
		return nil
	}
	return &model.ExchangeRate{
		ID:           e.ID,
		WorkspaceID:  e.WorkspaceID,
		FromCurrency: e.FromCurrency,
		ToCurrency:   e.ToCurrency,
		Rate:         e.Rate,
		RateDate:     e.RateDate,
		Source:       e.Source,
	}
}

func NewExchangeRateRepository(cl *ent.Client, log *slog.Logger) ExchangeRateRepository {
	return ExchangeRateRepository{
		cl:  cl,
		log: log,
	}
}

func (r ExchangeRateRepository) List(ctx context.Context, req model.ListExchangeRateRequest) ([]model.ExchangeRate, error) {
	q := r.cl.ExchangeRate.Query()

	if req.FromCurrency != "" {
		q.Where(exchangerate.FromCurrencyEQ(req.FromCurrency))
	}
	if req.ToCurrency != "" {
		q.Where(exchangerate.ToCurrencyEQ(req.ToCurrency))
	}
	if req.RateDate != "" {
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"} {
			if t, err := time.Parse(layout, req.RateDate); err == nil {
				q.Where(exchangerate.RateDateEQ(t))
				break
			}
		}
	}

	q.Order(ent.Desc(exchangerate.FieldRateDate))

	rates, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]model.ExchangeRate, len(rates))
	for i, rate := range rates {
		res[i] = *toExchangeRate(rate)
	}

	return res, nil
}

func (r ExchangeRateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ExchangeRate, error) {
	rate, err := r.cl.ExchangeRate.Query().Where(exchangerate.IDEQ(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, shared.ErrItemNotFound
		}
		return nil, err
	}

	return toExchangeRate(rate), nil
}

func (r ExchangeRateRepository) GetLatestRate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (*model.ExchangeRate, error) {
	if fromCurrency == toCurrency {
		return &model.ExchangeRate{
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			Rate:         decimal.NewFromInt(1),
			RateDate:     date,
		}, nil
	}

	rate, err := r.cl.ExchangeRate.Query().
		Where(
			exchangerate.FromCurrencyEQ(fromCurrency),
			exchangerate.ToCurrencyEQ(toCurrency),
			exchangerate.RateDateLTE(date),
		).
		Order(ent.Desc(exchangerate.FieldRateDate)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, shared.ErrItemNotFound
		}
		return nil, err
	}

	return toExchangeRate(rate), nil
}

func (r ExchangeRateRepository) Create(ctx context.Context, req model.CreateExchangeRateRequest) (*model.ExchangeRate, error) {
	builder := r.cl.ExchangeRate.Create().
		SetFromCurrency(req.FromCurrency).
		SetToCurrency(req.ToCurrency).
		SetRate(req.Rate).
		SetRateDate(req.RateDate)

	if req.Source != nil {
		builder.SetSource(*req.Source)
	}

	rate, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}

	return toExchangeRate(rate), nil
}

func (r ExchangeRateRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateExchangeRateRequest) (*model.ExchangeRate, error) {
	builder := r.cl.ExchangeRate.UpdateOneID(id)

	if req.Rate != nil {
		builder.SetRate(*req.Rate)
	}
	if req.Source != nil {
		builder.SetSource(*req.Source)
	}

	rate, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, shared.ErrItemNotFound
		}
		return nil, err
	}

	return toExchangeRate(rate), nil
}

func (r ExchangeRateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.cl.ExchangeRate.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return shared.ErrItemNotFound
		}
		return err
	}
	return nil
}
