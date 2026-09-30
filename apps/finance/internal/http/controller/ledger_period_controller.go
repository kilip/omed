package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/authz"
)

type LedgerPeriodService interface {
	List(ctx context.Context, req model.ListLedgerPeriodRequest) ([]model.LedgerPeriod, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.LedgerPeriod, error)
	Create(ctx context.Context, req model.CreateLedgerPeriodRequest) (*model.LedgerPeriod, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateLedgerPeriodRequest) (*model.LedgerPeriod, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type LedgerPeriodController struct {
	svc LedgerPeriodService
}

func NewLedgerPeriodController(svc LedgerPeriodService) *LedgerPeriodController {
	return &LedgerPeriodController{svc: svc}
}

func (h *LedgerPeriodController) Register(r fiber.Router) {
	g := r.Group("/ledger-periods")
	read := http.RequirePermission(authz.ResourceLedgerPeriods, authz.ActionRead)
	write := http.RequirePermission(authz.ResourceLedgerPeriods, authz.ActionWrite)

	g.Get("/", read, h.List)
	g.Get("/:id", read, h.Get)
	g.Post("/", write, h.Create)
	g.Put("/:id", write, h.Update)
	g.Delete("/:id", write, h.Delete)
}

// List godoc
//
//	@Summary		List ledger periods
//	@Description	Return all ledger periods in the active workspace.
//	@Tags			ledger-periods
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string	false	"Filter by status"	Enums(open, closed, locked)
//	@Param			date	query		string	false	"Filter periods covering this date"
//	@Success		200		{object}	model.WebResponse[[]model.LedgerPeriod]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/ledger-periods [get]
func (h *LedgerPeriodController) List(c fiber.Ctx) error {
	var req model.ListLedgerPeriodRequest
	if err := c.Bind().Query(&req); err != nil {
		return err
	}

	periods, err := h.svc.List(c, req)
	if err != nil {
		return err
	}
	return http.OK(c, periods)
}

// Get godoc
//
//	@Summary		Get ledger period
//	@Description	Get a single ledger period by ID.
//	@Tags			ledger-periods
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Ledger Period ID"	format(uuid)
//	@Success		200	{object}	model.WebResponse[model.LedgerPeriod]
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/ledger-periods/{id} [get]
func (h *LedgerPeriodController) Get(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	period, err := h.svc.GetByID(c, *id)
	if err != nil {
		return err
	}
	return http.OK(c, period)
}

// Create godoc
//
//	@Summary		Create ledger period
//	@Description	Create a new ledger period in the active workspace.
//	@Tags			ledger-periods
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		model.CreateLedgerPeriodRequest	true	"Ledger period payload"
//	@Success		201		{object}	model.WebResponse[model.LedgerPeriod]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/ledger-periods [post]
func (h *LedgerPeriodController) Create(c fiber.Ctx) error {
	var req model.CreateLedgerPeriodRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	period, err := h.svc.Create(c, req)
	if err != nil {
		return err
	}
	return http.Created(c, period)
}

// Update godoc
//
//	@Summary		Update ledger period status
//	@Description	Update status of a ledger period following transition: open -> closed -> locked.
//	@Tags			ledger-periods
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"Ledger Period ID"	format(uuid)
//	@Param			request	body		model.UpdateLedgerPeriodRequest	true	"Ledger period payload"
//	@Success		200		{object}	model.WebResponse[model.LedgerPeriod]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		404		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/ledger-periods/{id} [put]
func (h *LedgerPeriodController) Update(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	var req model.UpdateLedgerPeriodRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	period, err := h.svc.Update(c, *id, req)
	if err != nil {
		return err
	}
	return http.OK(c, period)
}

// Delete godoc
//
//	@Summary		Delete ledger period
//	@Description	Delete a ledger period by ID.
//	@Tags			ledger-periods
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Ledger Period ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		422	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/ledger-periods/{id} [delete]
func (h *LedgerPeriodController) Delete(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	if err := h.svc.Delete(c, *id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
