// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/currency"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// CurrencyRepository defines operations for querying global currencies and managing workspace base currency.
type CurrencyRepository struct {
	cli *ent.Client
}

// NewCurrencyRepository constructs a new CurrencyRepository instance.
func NewCurrencyRepository(cli *ent.Client) CurrencyRepository {
	return CurrencyRepository{cli: cli}
}

// List retrieves all available currencies ordered by currency code ascending.
func (r CurrencyRepository) List(c context.Context) ([]domain.Currency, error) {
	currencies, err := r.cli.Currency.Query().
		Order(ent.Asc(currency.FieldCode)).
		All(c)
	if err != nil {
		return nil, err
	}

	res := make([]domain.Currency, len(currencies))
	for i, cur := range currencies {
		res[i] = toCurrencyDomain(cur)
	}
	return res, nil
}

// GetByCode retrieves a currency by its 3-letter ISO code.
func (r CurrencyRepository) GetByCode(c context.Context, code string) (domain.Currency, error) {
	cur, err := r.cli.Currency.Query().
		Where(currency.CodeEQ(code)).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Currency{}, domain.ErrCurrencyNotFound
		}
		return domain.Currency{}, err
	}
	return toCurrencyDomain(cur), nil
}

// GetBaseCurrency retrieves the base currency configured for a workspace.
func (r CurrencyRepository) GetBaseCurrency(c context.Context, workspaceID int64) (string, error) {
	w, err := r.cli.Workspace.Get(c, workspaceID)
	if err != nil {
		return "IDR", err
	}
	if w.BaseCurrency == "" {
		return "IDR", nil
	}
	return w.BaseCurrency, nil
}

// UpdateWorkspaceBaseCurrency updates the base_currency setting on a workspace.
func (r CurrencyRepository) UpdateWorkspaceBaseCurrency(c context.Context, workspaceID int64, baseCurrency string) (model.WorkspaceBaseCurrencyResponse, error) {
	var resp model.WorkspaceBaseCurrencyResponse
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		w, err := tx.Workspace.UpdateOneID(workspaceID).
			SetBaseCurrency(baseCurrency).
			Save(c)
		if err != nil {
			return err
		}
		resp = model.WorkspaceBaseCurrencyResponse{
			ID:           w.ID,
			Name:         w.Name,
			BaseCurrency: w.BaseCurrency,
		}
		return nil
	})
	return resp, err
}

func toCurrencyDomain(c *ent.Currency) domain.Currency {
	return domain.Currency{
		ID:        c.ID,
		Code:      c.Code,
		Name:      c.Name,
		Symbol:    c.Symbol,
		Decimals:  c.Decimals,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
