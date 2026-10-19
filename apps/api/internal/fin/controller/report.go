// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// ReportServiceContract defines the operations required by ReportController.
type ReportServiceContract interface {
	TrialBalance(ctx context.Context, req model.TrialBalanceRequest) (model.TrialBalanceResponse, error)
}

// ReportController handles HTTP requests for financial reports.
type ReportController struct {
	reports ReportServiceContract
}

// NewReportController constructs a new ReportController.
func NewReportController(reports ReportServiceContract) ReportController {
	return ReportController{reports: reports}
}

// TrialBalance generates a trial balance report as of the specified date.
//
//	@Summary		Generate trial balance report
//	@Description	Calculate trial balance for all accounts in base currency as of the specified date with optional currency breakdown
//	@Tags			Finance Reports
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.TrialBalanceRequest	true	"Trial balance request parameters"
//	@Success		200		{object}	httpx.WebResponse[model.TrialBalanceResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/reports/trial-balance [post]
func (ctl ReportController) TrialBalance(c fiber.Ctx) error {
	var req model.TrialBalanceRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.reports.TrialBalance(c, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}
