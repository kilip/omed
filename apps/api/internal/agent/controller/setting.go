package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// SettingServiceContract defines the operations required by SettingController.
type SettingServiceContract interface {
	Get(ctx context.Context) (domain.Setting, error)
	Update(ctx context.Context, req model.UpdateSettingRequest) (domain.Setting, error)
}

// SettingController handles HTTP requests for workspace agent settings.
type SettingController struct {
	svc SettingServiceContract
}

// NewSettingController constructs a new SettingController backed by SettingServiceContract.
func NewSettingController(svc SettingServiceContract) SettingController {
	return SettingController{svc: svc}
}

func toSettingResponse(s domain.Setting) model.SettingResponse {
	return model.SettingResponse{
		FinancialEnabled:    s.FinancialEnabled,
		FinancialProviderID: s.FinancialProviderID,
	}
}

// Get retrieves workspace agent settings.
//
//	@Summary		Get agent settings
//	@Description	Retrieve agent configuration for the workspace
//	@Tags			Agent Settings
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[model.SettingResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/settings [get]
func (ctl SettingController) Get(c fiber.Ctx) error {
	setting, err := ctl.svc.Get(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, toSettingResponse(setting))
}

// Update updates workspace agent settings.
//
//	@Summary		Update agent settings
//	@Description	Update agent configuration for the workspace
//	@Tags			Agent Settings
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.UpdateSettingRequest	true	"Setting update payload"
//	@Success		200		{object}	httpx.WebResponse[model.SettingResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/settings [patch]
func (ctl SettingController) Update(c fiber.Ctx) error {
	var req model.UpdateSettingRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	setting, err := ctl.svc.Update(c, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toSettingResponse(setting))
}
