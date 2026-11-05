package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// PostServiceContract defines the operations required by PostController.
type PostServiceContract interface {
	Create(c context.Context, req model.CreatePostRequest) (domain.Post, error)
	GetByID(c context.Context, id int64, requestedLocale ...string) (domain.Post, error)
	Search(c context.Context, req model.SearchPostRequest) ([]domain.Post, string, error)
	Update(c context.Context, id int64, req model.UpdatePostRequest, locale ...string) (domain.Post, error)
	Delete(c context.Context, id int64) error
	Publish(c context.Context, id int64, locale ...string) (domain.Post, error)
	Schedule(c context.Context, id int64, req model.SchedulePostRequest, locale ...string) (domain.Post, error)
	Unpublish(c context.Context, id int64, locale ...string) (domain.Post, error)
	SaveTranslation(c context.Context, id int64, locale string, req model.SavePostTranslationRequest) (domain.Post, error)
	DeleteTranslation(c context.Context, id int64, locale string) error
}

// PostController handles HTTP requests for CMS post management.
type PostController struct {
	svc PostServiceContract
}

// NewPostController constructs a new PostController backed by PostServiceContract.
func NewPostController(svc PostServiceContract) PostController {
	return PostController{svc: svc}
}

// Create creates a new draft post.
//
//	@Summary		Create post
//	@Description	Create a new draft post in the workspace
//	@Tags			CMS Posts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			request	body		model.CreatePostRequest	true	"Post data"
//	@Success		201		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts [post]
func (ctl PostController) Create(c fiber.Ctx) error {
	var req model.CreatePostRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	if req.Title == "" {
		return domain.ErrTitleRequired
	}
	post, err := ctl.svc.Create(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toPostResponse(post))
}

// Search searches posts in the workspace with query and filters.
//
//	@Summary		Search posts
//	@Description	Search posts in the workspace with filters and cursor pagination
//	@Tags			CMS Posts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			request	body		model.SearchPostRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/search [post]
func (ctl PostController) Search(c fiber.Ctx) error {
	var req model.SearchPostRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.svc.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.PostResponse, len(items))
	for i, it := range items {
		out[i] = toPostResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// GetByID retrieves a post by ID.
//
//	@Summary		Get post by ID
//	@Description	Get post details by ID for the requested or default locale
//	@Tags			CMS Posts
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			id		path		int		true	"Post ID"
//	@Param			locale	query		string	false	"BCP 47 locale tag"
//	@Success		200		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id} [get]
func (ctl PostController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	locale := c.Query("locale")
	item, err := ctl.svc.GetByID(c, id, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPostResponse(item))
}

// Update updates an existing post.
//
//	@Summary		Update post
//	@Description	Update content and metadata of an existing post
//	@Tags			CMS Posts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			id		path		int						true	"Post ID"
//	@Param			locale	query		string					false	"BCP 47 locale tag"
//	@Param			request	body		model.UpdatePostRequest	true	"Post update payload"
//	@Success		200		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id} [put]
func (ctl PostController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdatePostRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	locale := c.Query("locale")
	item, err := ctl.svc.Update(c, id, req, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPostResponse(item))
}

// SaveTranslation creates or updates a translation for a post.
//
//	@Summary		Save post translation
//	@Description	Create or update localized translation for a post
//	@Tags			CMS Posts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			id		path		int								true	"Post ID"
//	@Param			locale	path		string							true	"BCP 47 locale tag"
//	@Param			request	body		model.SavePostTranslationRequest	true	"Translation payload"
//	@Success		200		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id}/translations/{locale} [put]
func (ctl PostController) SaveTranslation(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	locale := c.Params("locale")
	var req model.SavePostTranslationRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	if req.Title == "" {
		return domain.ErrTitleRequired
	}
	item, err := ctl.svc.SaveTranslation(c, id, locale, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPostResponse(item))
}

// DeleteTranslation removes a translation for a post.
//
//	@Summary		Delete post translation
//	@Description	Delete a translation for a post by locale
//	@Tags			CMS Posts
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			id		path		int		true	"Post ID"
//	@Param			locale	path		string	true	"BCP 47 locale tag"
//	@Success		200		{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id}/translations/{locale} [delete]
func (ctl PostController) DeleteTranslation(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	locale := c.Params("locale")
	if err := ctl.svc.DeleteTranslation(c, id, locale); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Delete soft-deletes a post.
//
//	@Summary		Delete post
//	@Description	Soft delete a post by ID
//	@Tags			CMS Posts
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Post ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id} [delete]
func (ctl PostController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.svc.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Publish publishes a post immediately.
//
//	@Summary		Publish post
//	@Description	Publish a draft or scheduled post immediately
//	@Tags			CMS Posts
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			id		path		int		true	"Post ID"
//	@Param			locale	query		string	false	"BCP 47 locale tag"
//	@Success		200		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id}/publish [post]
func (ctl PostController) Publish(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	locale := c.Query("locale")
	item, err := ctl.svc.Publish(c, id, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPostResponse(item))
}

// Schedule schedules a post for future publication.
//
//	@Summary		Schedule post
//	@Description	Schedule a post to be published at a future time
//	@Tags			CMS Posts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Post ID"
//	@Param			locale	query		string						false	"BCP 47 locale tag"
//	@Param			request	body		model.SchedulePostRequest	true	"Schedule details"
//	@Success		200		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id}/schedule [post]
func (ctl PostController) Schedule(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.SchedulePostRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	if req.PublishAt.IsZero() {
		return domain.ErrPublishAtRequired
	}
	locale := c.Query("locale")
	item, err := ctl.svc.Schedule(c, id, req, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPostResponse(item))
}

// Unpublish reverts a post back to draft status.
//
//	@Summary		Unpublish post
//	@Description	Unpublish a published or scheduled post back to draft
//	@Tags			CMS Posts
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			id		path		int		true	"Post ID"
//	@Param			locale	query		string	false	"BCP 47 locale tag"
//	@Success		200		{object}	httpx.WebResponse[model.PostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/posts/{id}/unpublish [post]
func (ctl PostController) Unpublish(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	locale := c.Query("locale")
	item, err := ctl.svc.Unpublish(c, id, locale)
	if err != nil {
		return err
	}
	return httpx.OK(c, toPostResponse(item))
}

func toPostResponse(p domain.Post) model.PostResponse {
	resp := model.PostResponse{
		ID:              p.ID,
		WorkspaceID:     p.WorkspaceID,
		Locale:          p.Locale,
		Title:           p.Title,
		Slug:            p.Slug,
		Excerpt:         p.Excerpt,
		Body:            p.Body,
		CoverMediaID:    p.CoverMediaID,
		Status:          p.Status,
		EffectiveStatus: p.EffectiveStatus,
		PublishAt:       p.PublishAt,
		Translations:    p.Translations,
		AuditMeta:       p.AuditMeta,
	}
	if p.CoverMedia != nil {
		cover := toMediaResponse(*p.CoverMedia)
		resp.CoverImage = &cover
	}
	return resp
}
