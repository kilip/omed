// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// ExchangeRateServiceContract defines the operations required by ExchangeRateController.
type ExchangeRateServiceContract interface {
	GetByID(c context.Context, id int64) (domain.ExchangeRate, error)
	Create(c context.Context, req model.CreateExchangeRateRequest) (domain.ExchangeRate, error)
	Update(c context.Context, id int64, req model.UpdateExchangeRateRequest) (domain.ExchangeRate, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchExchangeRateRequest) ([]domain.ExchangeRate, string, error)
	Lookup(c context.Context, from, to string, date time.Time) (float64, error)
}

// ExchangeRateController handles HTTP requests for Exchange Rates.
type ExchangeRateController struct {
	rates ExchangeRateServiceContract
}

// NewExchangeRateController constructs a new ExchangeRateController backed by ExchangeRateServiceContract.
func NewExchangeRateController(rates ExchangeRateServiceContract) ExchangeRateController {
	return ExchangeRateController{rates: rates}
}

// Search searches exchange rates within a workspace based on query parameters.
//
//	@Summary		Search exchange rates
//	@Description	Search exchange rates in the workspace
//	@Tags			Finance Exchange Rates
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.SearchExchangeRateRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.ExchangeRateResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/exchange-rates/search [post]
func (ctl ExchangeRateController) Search(c fiber.Ctx) error {
	var req model.SearchExchangeRateRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.rates.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.ExchangeRateResponse, len(items))
	for i, it := range items {
		out[i] = toExchangeRateResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// Create creates a new exchange rate in the workspace.
//
//	@Summary		Create exchange rate
//	@Description	Create a new exchange rate entry in the workspace
//	@Tags			Finance Exchange Rates
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.CreateExchangeRateRequest	true	"Exchange rate creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.ExchangeRateResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/exchange-rates [post]
func (ctl ExchangeRateController) Create(c fiber.Ctx) error {
	var req model.CreateExchangeRateRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.rates.Create(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toExchangeRateResponse(res))
}

// GetByID retrieves a single exchange rate by ID in the workspace.
//
//	@Summary		Get exchange rate by ID
//	@Description	Retrieve exchange rate detail by ID within the workspace
//	@Tags			Finance Exchange Rates
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Exchange Rate ID"
//	@Success		200	{object}	httpx.WebResponse[model.ExchangeRateResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/exchange-rates/{id} [get]
func (ctl ExchangeRateController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	res, err := ctl.rates.GetByID(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toExchangeRateResponse(res))
}

// Update modifies an existing exchange rate in the workspace.
//
//	@Summary		Update exchange rate
//	@Description	Update an existing exchange rate by ID
//	@Tags			Finance Exchange Rates
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			id		path		int								true	"Exchange Rate ID"
//	@Param			request	body		model.UpdateExchangeRateRequest	true	"Exchange rate update payload"
//	@Success		200		{object}	httpx.WebResponse[model.ExchangeRateResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/exchange-rates/{id} [put]
func (ctl ExchangeRateController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateExchangeRateRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.rates.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toExchangeRateResponse(res))
}

// Delete soft-deletes an exchange rate by ID.
//
//	@Summary		Delete exchange rate
//	@Description	Soft delete an exchange rate by ID
//	@Tags			Finance Exchange Rates
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Exchange Rate ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/exchange-rates/{id} [delete]
func (ctl ExchangeRateController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.rates.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Lookup finds the best exchange rate on or before a given date.
//
//	@Summary		Lookup exchange rate
//	@Description	Find the best exchange rate for a currency pair effective on or before the given date
//	@Tags			Finance Exchange Rates
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			from	query		string	true	"From currency code (e.g. USD)"
//	@Param			to		query		string	true	"To currency code (e.g. IDR)"
//	@Param			date	query		string	false	"Effective date (YYYY-MM-DD)"
//	@Success		200		{object}	httpx.WebResponse[model.ExchangeRateLookupResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/exchange-rates/lookup [get]
func (ctl ExchangeRateController) Lookup(c fiber.Ctx) error {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		return domain.ErrExchangeRateInvalid
	}

	dateStr := c.Query("date")
	targetDate := time.Now()
	if dateStr != "" {
		parsed := false
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05Z07:00"} {
			if t, err := time.Parse(layout, dateStr); err == nil {
				targetDate = t
				parsed = true
				break
			}
		}
		if !parsed {
			return fmt.Errorf("invalid date format: %s", dateStr)
		}
	}

	rate, err := ctl.rates.Lookup(c, from, to, targetDate)
	if err != nil {
		return err
	}

	return httpx.OK(c, model.ExchangeRateLookupResponse{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rate,
		Date:         targetDate,
	})
}

func toExchangeRateResponse(er domain.ExchangeRate) model.ExchangeRateResponse {
	return model.ExchangeRateResponse{
		ID:            er.ID,
		WorkspaceID:   er.WorkspaceID,
		FromCurrency:  er.FromCurrency,
		ToCurrency:    er.ToCurrency,
		Rate:          er.Rate,
		EffectiveDate: model.Date(er.EffectiveDate),
		Source:        string(er.Source),
		CreatedBy:     er.CreatedBy,
		CreatedAt:     er.CreatedAt,
		UpdatedBy:     er.UpdatedBy,
		UpdatedAt:     er.UpdatedAt,
	}
}
