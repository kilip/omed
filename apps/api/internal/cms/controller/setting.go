package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// SettingServiceContract defines operations required by SettingController.
type SettingServiceContract interface {
	Get(ctx context.Context) (domain.Setting, error)
	Update(ctx context.Context, req model.UpdateCmsSettingRequest) (domain.Setting, error)
}

// SettingController handles HTTP requests for CMS workspace settings.
type SettingController struct {
	svc SettingServiceContract
}

// NewSettingController constructs a new SettingController.
func NewSettingController(svc SettingServiceContract) SettingController {
	return SettingController{svc: svc}
}

func toSettingResponse(s domain.Setting) model.CmsSettingResponse {
	return model.CmsSettingResponse{
		DefaultLocale:    s.DefaultLocale,
		SupportedLocales: s.SupportedLocales,
	}
}

// Get retrieves CMS settings for the workspace.
//
//	@Summary		Get CMS settings
//	@Description	Retrieve CMS default and supported locales for the workspace
//	@Tags			CMS Settings
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[model.CmsSettingResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/settings [get]
func (ctl SettingController) Get(c fiber.Ctx) error {
	setting, err := ctl.svc.Get(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, toSettingResponse(setting))
}

// Update updates CMS settings for the workspace.
//
//	@Summary		Update CMS settings
//	@Description	Update CMS default and supported locales for the workspace
//	@Tags			CMS Settings
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.UpdateCmsSettingRequest	true	"Setting update payload"
//	@Success		200		{object}	httpx.WebResponse[model.CmsSettingResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/settings [put]
func (ctl SettingController) Update(c fiber.Ctx) error {
	var req model.UpdateCmsSettingRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	setting, err := ctl.svc.Update(c, req)
	if err != nil {
		return err
	}

	return httpx.OK(c, toSettingResponse(setting))
}
