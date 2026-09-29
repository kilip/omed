package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/authz"
)

type ExchangeRateService interface {
	List(ctx context.Context, req model.ListExchangeRateRequest) ([]model.ExchangeRate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.ExchangeRate, error)
	Create(ctx context.Context, req model.CreateExchangeRateRequest) (*model.ExchangeRate, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateExchangeRateRequest) (*model.ExchangeRate, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ExchangeRateController struct {
	svc ExchangeRateService
}

func NewExchangeRateController(svc ExchangeRateService) *ExchangeRateController {
	return &ExchangeRateController{svc: svc}
}

func (h *ExchangeRateController) Register(r fiber.Router) {
	g := r.Group("/exchange-rates")
	read := http.RequirePermission(authz.ResourceExchangeRates, authz.ActionRead)
	write := http.RequirePermission(authz.ResourceExchangeRates, authz.ActionWrite)

	g.Get("/", read, h.List)
	g.Get("/:id", read, h.Get)
	g.Post("/", write, h.Create)
	g.Put("/:id", write, h.Update)
	g.Delete("/:id", write, h.Delete)
}

// List godoc
//
//	@Summary		List exchange rates
//	@Description	Return exchange rates with optional filtering.
//	@Tags			exchange-rates
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from_currency	query		string	false	"Filter by base currency"
//	@Param			to_currency		query		string	false	"Filter by target currency"
//	@Param			rate_date		query		string	false	"Filter by effective date"
//	@Success		200				{object}	model.WebResponse[[]model.ExchangeRate]
//	@Failure		400				{object}	model.ErrorResponse
//	@Failure		401				{object}	model.ErrorResponse
//	@Failure		403				{object}	model.ErrorResponse
//	@Failure		422				{object}	model.ErrorResponse
//	@Failure		500				{object}	model.ErrorResponse
//	@Router			/exchange-rates [get]
func (h *ExchangeRateController) List(c fiber.Ctx) error {
	var req model.ListExchangeRateRequest
	if err := c.Bind().Query(&req); err != nil {
		return err
	}

	rates, err := h.svc.List(c, req)
	if err != nil {
		return err
	}
	return http.OK(c, rates)
}

// Get godoc
//
//	@Summary		Get exchange rate
//	@Description	Get a single exchange rate entry by ID.
//	@Tags			exchange-rates
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Exchange Rate ID"	format(uuid)
//	@Success		200	{object}	model.WebResponse[model.ExchangeRate]
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/exchange-rates/{id} [get]
func (h *ExchangeRateController) Get(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	rate, err := h.svc.GetByID(c, *id)
	if err != nil {
		return err
	}
	return http.OK(c, rate)
}

// Create godoc
//
//	@Summary		Create exchange rate
//	@Description	Create a new currency exchange rate.
//	@Tags			exchange-rates
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		model.CreateExchangeRateRequest	true	"Exchange rate payload"
//	@Success		201		{object}	model.WebResponse[model.ExchangeRate]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/exchange-rates [post]
func (h *ExchangeRateController) Create(c fiber.Ctx) error {
	var req model.CreateExchangeRateRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	rate, err := h.svc.Create(c, req)
	if err != nil {
		return err
	}
	return http.Created(c, rate)
}

// Update godoc
//
//	@Summary		Update exchange rate
//	@Description	Update rate or source of an exchange rate entry.
//	@Tags			exchange-rates
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"Exchange Rate ID"	format(uuid)
//	@Param			request	body		model.UpdateExchangeRateRequest	true	"Exchange rate payload"
//	@Success		200		{object}	model.WebResponse[model.ExchangeRate]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		404		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/exchange-rates/{id} [put]
func (h *ExchangeRateController) Update(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	var req model.UpdateExchangeRateRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	rate, err := h.svc.Update(c, *id, req)
	if err != nil {
		return err
	}
	return http.OK(c, rate)
}

// Delete godoc
//
//	@Summary		Delete exchange rate
//	@Description	Delete an exchange rate entry by ID.
//	@Tags			exchange-rates
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Exchange Rate ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		422	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/exchange-rates/{id} [delete]
func (h *ExchangeRateController) Delete(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	if err := h.svc.Delete(c, *id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
