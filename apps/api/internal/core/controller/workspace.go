// Package controller provides HTTP handlers and endpoint bindings for user and workspace management.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// WorkspaceService defines the business logic operations required by WorkspaceController.
type WorkspaceService interface {
	GetByID(c context.Context, id int64) (domain.Workspace, error)
	Search(c context.Context, r model.SearchWorkspaceRequest) ([]domain.Workspace, string, error)
	Create(c context.Context, r model.CreateWorkspaceRequest) (domain.Workspace, error)
	Update(c context.Context, id int64, r model.UpdateWorkspaceRequest) (domain.Workspace, error)
	Delete(c context.Context, id int64) error
}

// WorkspaceController handles HTTP requests for workspace CRUD operations and search.
type WorkspaceController struct {
	workspaces WorkspaceService
}

// NewWorkspaceController constructs a new WorkspaceController backed by the specified WorkspaceService.
func NewWorkspaceController(workspaces WorkspaceService) WorkspaceController {
	return WorkspaceController{workspaces}
}

// GetByID gets a workspace by ID.
//
//	@Summary		Get workspace by ID
//	@Description	Retrieve workspace detail by ID
//	@Tags			Workspaces
//	@Produce		json
//	@Param			id	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[model.WorkspaceResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{id} [get]
func (w WorkspaceController) GetByID(c fiber.Ctx) error {
	return httpx.DoGetByID(c, w.workspaces.GetByID, toWorkspaceResponse)
}

// Search searches workspaces based on query parameters.
//
//	@Summary		Search workspaces
//	@Description	Search workspaces matching filter criteria with pagination cursor
//	@Tags			Workspaces
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.SearchWorkspaceRequest	true	"Search filter criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.WorkspaceResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/search [post]
func (w WorkspaceController) Search(c fiber.Ctx) error {
	return httpx.DoSearch(c, w.workspaces.Search, toWorkspaceResponse)
}

// Create creates a new workspace.
//
//	@Summary		Create workspace
//	@Description	Create a new workspace
//	@Tags			Workspaces
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.CreateWorkspaceRequest	true	"Workspace creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.WorkspaceResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces [post]
func (w WorkspaceController) Create(c fiber.Ctx) error {
	return httpx.DoCreate(c, w.workspaces.Create, toWorkspaceResponse)
}

// Update updates an existing workspace by ID.
//
//	@Summary		Update workspace
//	@Description	Update an existing workspace by ID
//	@Tags			Workspaces
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"Workspace ID"
//	@Param			request	body		model.UpdateWorkspaceRequest	true	"Workspace update payload"
//	@Success		200		{object}	httpx.WebResponse[model.WorkspaceResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{id} [put]
func (w WorkspaceController) Update(c fiber.Ctx) error {
	return httpx.DoUpdate(c, w.workspaces.Update, toWorkspaceResponse)
}

// Delete deletes a workspace by ID.
//
//	@Summary		Delete workspace
//	@Description	Delete a workspace by ID
//	@Tags			Workspaces
//	@Produce		json
//	@Param			id	path	int	true	"Workspace ID"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{id} [delete]
func (w WorkspaceController) Delete(c fiber.Ctx) error {
	return httpx.DoDelete(c, w.workspaces.Delete)
}

func toWorkspaceResponse(ws domain.Workspace) model.WorkspaceResponse {
	return model.WorkspaceResponse{
		ID:          ws.ID,
		Name:        ws.Name,
		Description: ws.Description,
		Owner: model.UserResponse{
			ID:        ws.Owner.ID,
			Name:      ws.Owner.Name,
			Avatar:    ws.Owner.Avatar,
			CreatedAt: ws.Owner.CreatedAt,
			UpdatedAt: ws.Owner.UpdatedAt,
		},
		CreatedBy: ws.CreatedBy,
		CreatedAt: ws.CreatedAt,
		UpdatedBy: ws.UpdatedBy,
		UpdatedAt: ws.UpdatedAt,
	}
}

