// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/exchangerate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// ExchangeRateRepository defines persistence operations for exchange rates in a workspace.
type ExchangeRateRepository struct {
	cli *ent.Client
}

// NewExchangeRateRepository constructs a new ExchangeRateRepository instance backed by the given Ent client.
func NewExchangeRateRepository(cli *ent.Client) ExchangeRateRepository {
	return ExchangeRateRepository{cli: cli}
}

// GetByID retrieves an active exchange rate by its ID within a workspace.
func (r ExchangeRateRepository) GetByID(c context.Context, id int64) (domain.ExchangeRate, error) {
	var dom domain.ExchangeRate

	er, err := r.cli.ExchangeRate.Query().
		Where(
			exchangerate.ID(id),
			exchangerate.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, base.ErrNotFound
		}
		return dom, err
	}

	return toExchangeRateDomain(er), nil
}

// Create inserts a new exchange rate within a database transaction.
func (r ExchangeRateRepository) Create(c context.Context, req model.CreateExchangeRateRequest) (domain.ExchangeRate, error) {
	var dom domain.ExchangeRate

	if req.Rate <= 0 {
		return dom, domain.ErrExchangeRateInvalid
	}

	var createdID int64
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		effDate := req.EffectiveDate.Time()

		exists, err := tx.ExchangeRate.Query().
			Where(
				exchangerate.FromCurrencyEQ(req.FromCurrency),
				exchangerate.ToCurrencyEQ(req.ToCurrency),
				exchangerate.EffectiveDateEQ(effDate),
				exchangerate.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrExchangeRateDuplicate
		}

		src := exchangerate.SourceMANUAL
		if req.Source != nil && *req.Source != "" {
			src = exchangerate.Source(*req.Source)
		}

		res, err := tx.ExchangeRate.Create().
			SetFromCurrency(req.FromCurrency).
			SetToCurrency(req.ToCurrency).
			SetRate(req.Rate).
			SetEffectiveDate(effDate).
			SetSource(src).
			Save(c)
		if err != nil {
			return err
		}
		createdID = res.ID
		return nil
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, createdID)
}

// Update modifies an existing exchange rate within a database transaction.
func (r ExchangeRateRepository) Update(c context.Context, id int64, req model.UpdateExchangeRateRequest) (domain.ExchangeRate, error) {
	var dom domain.ExchangeRate

	if req.Rate <= 0 {
		return dom, domain.ErrExchangeRateInvalid
	}

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		existing, err := tx.ExchangeRate.Query().
			Where(
				exchangerate.ID(id),
				exchangerate.DeletedAtIsNil(),
			).
			Only(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return base.ErrNotFound
			}
			return err
		}

		effDate := req.EffectiveDate.Time()
		if !effDate.Equal(existing.EffectiveDate) {
			exists, err := tx.ExchangeRate.Query().
				Where(
					exchangerate.IDNEQ(id),
					exchangerate.FromCurrencyEQ(existing.FromCurrency),
					exchangerate.ToCurrencyEQ(existing.ToCurrency),
					exchangerate.EffectiveDateEQ(effDate),
					exchangerate.DeletedAtIsNil(),
				).
				Exist(c)
			if err != nil {
				return err
			}
			if exists {
				return domain.ErrExchangeRateDuplicate
			}
		}

		u := tx.ExchangeRate.UpdateOneID(id).
			SetRate(req.Rate).
			SetEffectiveDate(effDate)

		if req.Source != nil && *req.Source != "" {
			u.SetSource(exchangerate.Source(*req.Source))
		}

		return u.Exec(c)
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Delete marks an existing exchange rate as soft-deleted.
func (r ExchangeRateRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.ExchangeRate.Query().
			Where(
				exchangerate.ID(id),
				exchangerate.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		now := time.Now()
		return tx.ExchangeRate.UpdateOneID(id).
			SetDeletedAt(now).
			Exec(c)
	})
}

// Search searches active exchange rates matching filter criteria with pagination.
func (r ExchangeRateRepository) Search(c context.Context, req model.SearchExchangeRateRequest) ([]domain.ExchangeRate, string, error) {
	q := r.cli.ExchangeRate.Query().
		Where(
			exchangerate.DeletedAtIsNil(),
		)

	if req.FromCurrency != nil && *req.FromCurrency != "" {
		q = q.Where(exchangerate.FromCurrencyEQ(*req.FromCurrency))
	}
	if req.ToCurrency != nil && *req.ToCurrency != "" {
		q = q.Where(exchangerate.ToCurrencyEQ(*req.ToCurrency))
	}
	if req.StartDate != nil && !req.StartDate.IsZero() {
		q = q.Where(exchangerate.EffectiveDateGTE(req.StartDate.Time()))
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		q = q.Where(exchangerate.EffectiveDateLTE(req.EndDate.Time()))
	}

	items, err := q.
		Order(ent.Desc(exchangerate.FieldEffectiveDate), ent.Desc(exchangerate.FieldID)).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	res := make([]domain.ExchangeRate, len(items))
	for i, it := range items {
		res[i] = toExchangeRateDomain(it)
	}

	return res, "", nil
}

// Lookup finds the latest exchange rate effective on or before the given date.
// If a direct pair (from -> to) is found, returns the rate.
// If an inverse pair (to -> from) is found, returns 1.0 / rate.
// If neither is found, returns domain.ErrExchangeRateNotFound.
func (r ExchangeRateRepository) Lookup(c context.Context, from, to string, date time.Time) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	// 1. Direct pair lookup
	direct, err := r.cli.ExchangeRate.Query().
		Where(
			exchangerate.FromCurrencyEQ(from),
			exchangerate.ToCurrencyEQ(to),
			exchangerate.EffectiveDateLTE(date),
			exchangerate.DeletedAtIsNil(),
		).
		Order(ent.Desc(exchangerate.FieldEffectiveDate), ent.Desc(exchangerate.FieldID)).
		First(c)
	if err == nil {
		if direct.Rate <= 0 {
			return 0, domain.ErrExchangeRateInvalid
		}
		return direct.Rate, nil
	}
	if !ent.IsNotFound(err) {
		return 0, err
	}

	// 2. Inverse pair lookup
	inverse, err := r.cli.ExchangeRate.Query().
		Where(
			exchangerate.FromCurrencyEQ(to),
			exchangerate.ToCurrencyEQ(from),
			exchangerate.EffectiveDateLTE(date),
			exchangerate.DeletedAtIsNil(),
		).
		Order(ent.Desc(exchangerate.FieldEffectiveDate), ent.Desc(exchangerate.FieldID)).
		First(c)
	if err == nil {
		if inverse.Rate <= 0 {
			return 0, domain.ErrExchangeRateInvalid
		}
		return 1.0 / inverse.Rate, nil
	}
	if !ent.IsNotFound(err) {
		return 0, err
	}

	return 0, domain.ErrExchangeRateNotFound
}

func toExchangeRateDomain(er *ent.ExchangeRate) domain.ExchangeRate {
	dom := domain.ExchangeRate{
		ID:            er.ID,
		WorkspaceID:   er.WorkspaceID,
		FromCurrency:  er.FromCurrency,
		ToCurrency:    er.ToCurrency,
		Rate:          er.Rate,
		EffectiveDate: er.EffectiveDate,
		Source:        domain.ExchangeRateSource(er.Source),
		CreatedByID:   er.CreatedBy,
		UpdatedByID:   er.UpdatedBy,
		CreatedAt:     er.CreatedAt,
		UpdatedAt:     er.UpdatedAt,
	}

	if er.DeletedAt != nil {
		dom.DeletedAt = er.DeletedAt
	}
	if er.Edges.Creator != nil {
		dom.CreatedBy = base.ByUser{
			Name:   er.Edges.Creator.Name,
			Avatar: er.Edges.Creator.Avatar,
		}
	}
	if er.Edges.Updater != nil {
		dom.UpdatedBy = base.ByUser{
			Name:   er.Edges.Updater.Name,
			Avatar: er.Edges.Updater.Avatar,
		}
	}

	return dom
}
