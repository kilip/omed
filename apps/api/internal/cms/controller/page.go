package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// PageServiceContract defines the operations required by PageController.
type PageServiceContract interface {
	Create(c context.Context, req model.CreatePageRequest) (domain.Page, error)
	GetByID(c context.Context, id int64) (domain.Page, error)
	Search(c context.Context, req model.SearchPageRequest) ([]domain.Page, string, error)
	Update(c context.Context, id int64, req model.UpdatePageRequest) (domain.Page, error)
	Delete(c context.Context, id int64) error
	Publish(c context.Context, id int64) (domain.Page, error)
	Schedule(c context.Context, id int64, req model.SchedulePageRequest) (domain.Page, error)
	Unpublish(c context.Context, id int64) (domain.Page, error)
}

// PageController handles HTTP requests for CMS page management.
type PageController struct {
	svc PageServiceContract
}

// NewPageController constructs a new PageController backed by PageServiceContract.
func NewPageController(svc PageServiceContract) PageController {
	return PageController{svc: svc}
}

// Create creates a new draft page.
//
//	@Summary		Create page
//	@Description	Create a new draft page in the workspace
//	@Tags			CMS Pages
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			request	body		model.CreatePageRequest	true	"Page data"
//	@Success		201		{object}	httpx.WebResponse[model.PageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages [post]
func (ctl PageController) Create(c fiber.Ctx) error {
	var req model.CreatePageRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	if req.Title == "" {
		return domain.ErrTitleRequired
	}
	page, err := ctl.svc.Create(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toPageResponse(page))
}

// Search searches pages in the workspace with query and filters.
//
//	@Summary		Search pages
//	@Description	Search pages in the workspace with filters and cursor pagination
//	@Tags			CMS Pages
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			request	body		model.SearchPageRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.PageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/search [post]
func (ctl PageController) Search(c fiber.Ctx) error {
	var req model.SearchPageRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.svc.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.PageResponse, len(items))
	for i, it := range items {
		out[i] = toPageResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// GetByID retrieves a page by ID.
//
//	@Summary		Get page by ID
//	@Description	Get page details by ID
//	@Tags			CMS Pages
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Page ID"
//	@Success		200	{object}	httpx.WebResponse[model.PageResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/{id} [get]
func (ctl PageController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	item, err := ctl.svc.GetByID(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPageResponse(item))
}

// Update updates an existing page.
//
//	@Summary		Update page
//	@Description	Update content and metadata of an existing page
//	@Tags			CMS Pages
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			id		path		int						true	"Page ID"
//	@Param			request	body		model.UpdatePageRequest	true	"Page update payload"
//	@Success		200		{object}	httpx.WebResponse[model.PageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/{id} [put]
func (ctl PageController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdatePageRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	item, err := ctl.svc.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPageResponse(item))
}

// Delete soft-deletes a page.
//
//	@Summary		Delete page
//	@Description	Soft delete a page by ID
//	@Tags			CMS Pages
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Page ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/{id} [delete]
func (ctl PageController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.svc.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Publish publishes a page immediately.
//
//	@Summary		Publish page
//	@Description	Publish a draft or scheduled page immediately
//	@Tags			CMS Pages
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Page ID"
//	@Success		200	{object}	httpx.WebResponse[model.PageResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/{id}/publish [post]
func (ctl PageController) Publish(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	item, err := ctl.svc.Publish(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPageResponse(item))
}

// Schedule schedules a page for future publication.
//
//	@Summary		Schedule page
//	@Description	Schedule a page to be published at a future time
//	@Tags			CMS Pages
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Page ID"
//	@Param			request	body		model.SchedulePageRequest	true	"Schedule details"
//	@Success		200		{object}	httpx.WebResponse[model.PageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/{id}/schedule [post]
func (ctl PageController) Schedule(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.SchedulePageRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	if req.PublishAt.IsZero() {
		return domain.ErrPublishAtRequired
	}
	item, err := ctl.svc.Schedule(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPageResponse(item))
}

// Unpublish reverts a page back to draft status.
//
//	@Summary		Unpublish page
//	@Description	Unpublish a published or scheduled page back to draft
//	@Tags			CMS Pages
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Page ID"
//	@Success		200	{object}	httpx.WebResponse[model.PageResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/pages/{id}/unpublish [post]
func (ctl PageController) Unpublish(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	item, err := ctl.svc.Unpublish(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPageResponse(item))
}

func toPageResponse(p domain.Page) model.PageResponse {
	return model.PageResponse{
		ID:              p.ID,
		WorkspaceID:     p.WorkspaceID,
		Title:           p.Title,
		Slug:            p.Slug,
		Body:            p.Body,
		Status:          p.Status,
		EffectiveStatus: p.EffectiveStatus,
		PublishAt:       p.PublishAt,
		AuditMeta:       p.AuditMeta,
	}
}
