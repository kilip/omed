// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// SettingServiceContract defines the operations required by SettingController.
type SettingServiceContract interface {
	GetFXAccounts(c context.Context, workspaceID int64) (model.FXAccountsSettingResponse, error)
	UpdateFXAccounts(c context.Context, workspaceID int64, req model.UpdateFXAccountsRequest) (model.FXAccountsSettingResponse, error)
}

// SettingController handles HTTP requests for workspace financial settings.
type SettingController struct {
	settings SettingServiceContract
}

// NewSettingController constructs a new SettingController backed by SettingServiceContract.
func NewSettingController(settings SettingServiceContract) SettingController {
	return SettingController{settings: settings}
}

// GetFXAccounts retrieves the FX account mappings for the workspace.
//
//	@Summary		Get FX account mappings
//	@Description	Retrieve the configured foreign exchange accounts for the workspace
//	@Tags			Finance Settings
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[model.FXAccountsSettingResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/settings/fx-accounts [get]
func (ctl SettingController) GetFXAccounts(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	res, err := ctl.settings.GetFXAccounts(c, wid)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

// UpdateFXAccounts updates the FX account mappings for the workspace.
//
//	@Summary		Update FX account mappings
//	@Description	Configure the foreign exchange accounts for the workspace
//	@Tags			Finance Settings
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.UpdateFXAccountsRequest	true	"FX accounts payload"
//	@Success		200		{object}	httpx.WebResponse[model.FXAccountsSettingResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/settings/fx-accounts [put]
func (ctl SettingController) UpdateFXAccounts(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.UpdateFXAccountsRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.settings.UpdateFXAccounts(c, wid, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}
