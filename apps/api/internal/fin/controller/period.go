// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// PeriodServiceContract defines the operations required by PeriodController.
type PeriodServiceContract interface {
	GetByID(c context.Context, id int64) (domain.Period, error)
	Search(c context.Context, req model.SearchPeriodRequest) ([]domain.Period, string, error)
	Create(c context.Context, req model.CreatePeriodRequest) (domain.Period, error)
	Update(c context.Context, id int64, req model.UpdatePeriodRequest) (domain.Period, error)
	Delete(c context.Context, id int64) error
	Close(c context.Context, id int64) (domain.Period, error)
	Reopen(c context.Context, id int64) (domain.Period, error)
}

// PeriodController handles HTTP requests for Accounting Periods.
type PeriodController struct {
	periods PeriodServiceContract
}

// NewPeriodController constructs a new PeriodController backed by PeriodServiceContract.
func NewPeriodController(periods PeriodServiceContract) PeriodController {
	return PeriodController{periods: periods}
}

// Search searches accounting periods within a workspace based on query parameters.
//
//	@Summary		Search periods
//	@Description	Search accounting periods in the workspace
//	@Tags			Finance Periods
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SearchPeriodRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.PeriodResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods/search [post]
func (ctl PeriodController) Search(c fiber.Ctx) error {
	var req model.SearchPeriodRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.periods.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.PeriodResponse, len(items))
	for i, it := range items {
		out[i] = toPeriodResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// Create creates a new accounting period in the workspace.
//
//	@Summary		Create period
//	@Description	Create a new accounting period entry in the workspace
//	@Tags			Finance Periods
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.CreatePeriodRequest	true	"Period creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.PeriodResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods [post]
func (ctl PeriodController) Create(c fiber.Ctx) error {
	var req model.CreatePeriodRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	p, err := ctl.periods.Create(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toPeriodResponse(p))
}

// GetByID retrieves a single accounting period by ID in the workspace.
//
//	@Summary		Get period by ID
//	@Description	Retrieve accounting period detail by ID within the workspace
//	@Tags			Finance Periods
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Period ID"
//	@Success		200	{object}	httpx.WebResponse[model.PeriodResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods/{id} [get]
func (ctl PeriodController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	p, err := ctl.periods.GetByID(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPeriodResponse(p))
}

// Update updates an existing accounting period by ID in the workspace.
//
//	@Summary		Update period
//	@Description	Update an existing accounting period by ID within the workspace
//	@Tags			Finance Periods
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Period ID"
//	@Param			request	body		model.UpdatePeriodRequest	true	"Period update payload"
//	@Success		200		{object}	httpx.WebResponse[model.PeriodResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods/{id} [put]
func (ctl PeriodController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdatePeriodRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	p, err := ctl.periods.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPeriodResponse(p))
}

// Delete soft deletes an accounting period by ID.
//
//	@Summary		Delete period
//	@Description	Soft delete an accounting period by ID within the workspace
//	@Tags			Finance Periods
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Period ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods/{id} [delete]
func (ctl PeriodController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.periods.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Close closes an accounting period by ID.
//
//	@Summary		Close period
//	@Description	Close an accounting period, preventing new journal postings
//	@Tags			Finance Periods
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Period ID"
//	@Success		200	{object}	httpx.WebResponse[model.PeriodResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods/{id}/close [post]
func (ctl PeriodController) Close(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	p, err := ctl.periods.Close(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPeriodResponse(p))
}

// Reopen reopens a previously closed accounting period.
//
//	@Summary		Reopen period
//	@Description	Reopen a closed accounting period, allowing journal entries again
//	@Tags			Finance Periods
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Period ID"
//	@Success		200	{object}	httpx.WebResponse[model.PeriodResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/periods/{id}/reopen [post]
func (ctl PeriodController) Reopen(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	p, err := ctl.periods.Reopen(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPeriodResponse(p))
}

func toPeriodResponse(p domain.Period) model.PeriodResponse {
	return model.PeriodResponse{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		StartDate:   model.Date(p.StartDate),
		EndDate:     model.Date(p.EndDate),
		FiscalYear:  p.FiscalYear,
		Status:      p.Status,
		Description: p.Description,
		AuditMeta:   p.AuditMeta,
	}
}
