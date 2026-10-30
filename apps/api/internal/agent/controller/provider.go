package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// ProviderServiceContract defines the operations required by ProviderController.
type ProviderServiceContract interface {
	List(ctx context.Context) ([]domain.Provider, error)
	Get(ctx context.Context, id int64) (domain.Provider, error)
	Create(ctx context.Context, req model.CreateProviderRequest) (domain.Provider, error)
	Update(ctx context.Context, id int64, req model.UpdateProviderRequest) (domain.Provider, error)
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) (domain.Provider, error)
	Test(ctx context.Context, id int64) (model.TestResult, error)
	TestConfig(ctx context.Context, req model.TestProviderRequest) (model.TestResult, error)
}

// ProviderController handles HTTP requests for agent provider management.
type ProviderController struct {
	svc ProviderServiceContract
}

// NewProviderController constructs a new ProviderController backed by ProviderServiceContract.
func NewProviderController(svc ProviderServiceContract) ProviderController {
	return ProviderController{svc: svc}
}

// List lists all agent providers in the workspace.
//
//	@Summary		List providers
//	@Description	List all configured LLM providers in the workspace
//	@Tags			Agent Providers
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[[]model.ProviderResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers [get]
func (ctl ProviderController) List(c fiber.Ctx) error {
	items, err := ctl.svc.List(c)
	if err != nil {
		return err
	}
	out := make([]model.ProviderResponse, len(items))
	for i, item := range items {
		out[i] = toProviderResponse(item)
	}
	return httpx.OK(c, out)
}

// Get retrieves an agent provider by ID.
//
//	@Summary		Get provider by ID
//	@Description	Get agent provider details by ID
//	@Tags			Agent Providers
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Provider ID"
//	@Success		200	{object}	httpx.WebResponse[model.ProviderResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers/{id} [get]
func (ctl ProviderController) Get(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	item, err := ctl.svc.Get(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toProviderResponse(item))
}

// Create creates a new agent provider in the workspace.
//
//	@Summary		Create provider
//	@Description	Create a new agent provider configuration in the workspace
//	@Tags			Agent Providers
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.CreateProviderRequest	true	"Provider creation data"
//	@Success		201		{object}	httpx.WebResponse[model.ProviderResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Failure		503		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers [post]
func (ctl ProviderController) Create(c fiber.Ctx) error {
	var req model.CreateProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	item, err := ctl.svc.Create(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toProviderResponse(item))
}

// Update updates an existing agent provider.
//
//	@Summary		Update provider
//	@Description	Update agent provider configuration
//	@Tags			Agent Providers
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Provider ID"
//	@Param			request	body		model.UpdateProviderRequest	true	"Provider update payload"
//	@Success		200		{object}	httpx.WebResponse[model.ProviderResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Failure		503		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers/{id} [patch]
func (ctl ProviderController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	item, err := ctl.svc.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toProviderResponse(item))
}

// Delete deletes an agent provider.
//
//	@Summary		Delete provider
//	@Description	Delete an agent provider by ID
//	@Tags			Agent Providers
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Provider ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers/{id} [delete]
func (ctl ProviderController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.svc.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// SetDefault sets an agent provider as default for the workspace.
//
//	@Summary		Set default provider
//	@Description	Set an agent provider as default
//	@Tags			Agent Providers
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Provider ID"
//	@Success		200	{object}	httpx.WebResponse[model.ProviderResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers/{id}/default [post]
func (ctl ProviderController) SetDefault(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	item, err := ctl.svc.SetDefault(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toProviderResponse(item))
}

// Test checks connectivity and credentials of an existing agent provider.
//
//	@Summary		Test existing provider
//	@Description	Test connectivity and validity of an existing agent provider
//	@Tags			Agent Providers
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Provider ID"
//	@Success		200	{object}	httpx.WebResponse[model.TestResult]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Failure		503	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers/{id}/test [post]
func (ctl ProviderController) Test(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	res, err := ctl.svc.Test(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

// TestConfig checks connectivity and credentials for an unsaved provider configuration.
//
//	@Summary		Test provider config
//	@Description	Test connectivity and validity of unsaved agent provider configuration
//	@Tags			Agent Providers
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.TestProviderRequest	true	"Provider test config payload"
//	@Success		200		{object}	httpx.WebResponse[model.TestResult]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/providers/test [post]
func (ctl ProviderController) TestConfig(c fiber.Ctx) error {
	var req model.TestProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.svc.TestConfig(c, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

func toProviderResponse(p domain.Provider) model.ProviderResponse {
	return model.ProviderResponse{
		ID:           p.ID,
		Name:         p.Name,
		Kind:         string(p.Kind),
		BaseURL:      p.BaseURL,
		Model:        p.Model,
		HasKey:       p.HasKey,
		KeyHint:      p.KeyHint,
		IsDefault:    p.IsDefault,
		Capabilities: p.Capabilities,
		AuditMeta:    p.AuditMeta,
	}
}
