// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// CurrencyServiceContract defines the operations required by CurrencyController.
type CurrencyServiceContract interface {
	ListCurrencies(c context.Context) ([]domain.Currency, error)
	UpdateBaseCurrency(c context.Context, workspaceID int64, req model.UpdateBaseCurrencyRequest) (model.WorkspaceBaseCurrencyResponse, error)
}

// CurrencyController handles HTTP requests for currencies and workspace base currency configuration.
type CurrencyController struct {
	currencies CurrencyServiceContract
}

// NewCurrencyController constructs a new CurrencyController backed by CurrencyServiceContract.
func NewCurrencyController(currencies CurrencyServiceContract) CurrencyController {
	return CurrencyController{currencies: currencies}
}

// List lists all supported currencies.
//
//	@Summary		List currencies
//	@Description	Retrieve all supported reference currencies conforming to ISO 4217
//	@Tags			Finance Currencies
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[[]model.CurrencyResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/currencies [get]
func (ctl CurrencyController) List(c fiber.Ctx) error {
	items, err := ctl.currencies.ListCurrencies(c)
	if err != nil {
		return err
	}
	out := make([]model.CurrencyResponse, len(items))
	for i, it := range items {
		out[i] = model.CurrencyResponse{
			ID:       it.ID,
			Code:     it.Code,
			Name:     it.Name,
			Symbol:   it.Symbol,
			Decimals: it.Decimals,
		}
	}
	return httpx.OK(c, out)
}

// UpdateBaseCurrency updates the base currency setting for the workspace.
//
//	@Summary		Update workspace base currency
//	@Description	Update workspace base currency setting. Rejected if posted entries exist.
//	@Tags			Finance Settings
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.UpdateBaseCurrencyRequest	true	"Base currency payload"
//	@Success		200		{object}	httpx.WebResponse[model.WorkspaceBaseCurrencyResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/settings/base-currency [put]
func (ctl CurrencyController) UpdateBaseCurrency(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.UpdateBaseCurrencyRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.currencies.UpdateBaseCurrency(c, wid, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}
