package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// PublicPostServiceContract defines public post retrieval operations.
type PublicPostServiceContract interface {
	PublicList(c context.Context, workspaceID int64, cursor *string, limit int) ([]domain.Post, string, error)
	PublicGetBySlug(c context.Context, workspaceID int64, slug string) (domain.Post, error)
}

// PublicPageServiceContract defines public page retrieval operations.
type PublicPageServiceContract interface {
	PublicList(c context.Context, workspaceID int64, cursor *string, limit int) ([]domain.Page, string, error)
	PublicGetBySlug(c context.Context, workspaceID int64, slug string) (domain.Page, error)
}

// PublicMediaServiceContract defines the public media presigned URL retrieval.
type PublicMediaServiceContract interface {
	GetPublicPresignedURL(c context.Context, workspaceID, id int64) (string, error)
}

// PublicController handles public content retrieval endpoints.
type PublicController struct {
	postSvc  PublicPostServiceContract
	pageSvc  PublicPageServiceContract
	mediaSvc PublicMediaServiceContract
	apiURL   string
}

// NewPublicController constructs a new PublicController instance.
func NewPublicController(
	postSvc PublicPostServiceContract,
	pageSvc PublicPageServiceContract,
	mediaSvc PublicMediaServiceContract,
	apiURL string,
) PublicController {
	return PublicController{
		postSvc:  postSvc,
		pageSvc:  pageSvc,
		mediaSvc: mediaSvc,
		apiURL:   apiURL,
	}
}

// ListPosts lists publicly visible posts for a workspace.
//
//	@Summary		List public posts
//	@Description	List published posts visible at the current time for a workspace
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			cursor	query		string	false	"Composite pagination cursor"
//	@Param			limit	query		int		false	"Limit (1-100, default 20)"
//	@Success		200		{object}	httpx.WebResponse[[]model.PublicPostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/posts [get]
func (ctl PublicController) ListPosts(c fiber.Ctx) error {
	wid, err := strconv.ParseInt(c.Params("wid"), 10, 64)
	if err != nil || wid <= 0 {
		return httpx.WithCursor(c, []model.PublicPostResponse{}, "")
	}

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	limit := 20
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if l, err := strconv.Atoi(rawLimit); err == nil && l > 0 {
			limit = l
		}
	}

	var cursor *string
	if rawCursor := c.Query("cursor"); rawCursor != "" {
		cursor = &rawCursor
	}

	items, nextCursor, err := ctl.postSvc.PublicList(ctx, wid, cursor, limit)
	if err != nil {
		return err
	}

	out := make([]model.PublicPostResponse, len(items))
	for i, it := range items {
		out[i] = toPublicPostResponse(it, ctl.apiURL)
	}

	return httpx.WithCursor(c, out, nextCursor)
}

// GetPostBySlug retrieves a publicly visible post by slug.
//
//	@Summary		Get public post by slug
//	@Description	Retrieve a published post visible at current time by slug
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			slug	path		string	true	"Post slug"
//	@Success		200		{object}	httpx.WebResponse[model.PublicPostResponse]
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/posts/{slug} [get]
func (ctl PublicController) GetPostBySlug(c fiber.Ctx) error {
	wid, err := strconv.ParseInt(c.Params("wid"), 10, 64)
	if err != nil || wid <= 0 {
		return domain.ErrPostNotFound
	}
	slug := c.Params("slug")
	if slug == "" {
		return domain.ErrPostNotFound
	}

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	post, err := ctl.postSvc.PublicGetBySlug(ctx, wid, slug)
	if err != nil {
		return domain.ErrPostNotFound
	}

	return httpx.OK(c, toPublicPostResponse(post, ctl.apiURL))
}

// ListPages lists publicly visible pages for a workspace.
//
//	@Summary		List public pages
//	@Description	List published pages visible at the current time for a workspace
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			cursor	query		string	false	"Composite pagination cursor"
//	@Param			limit	query		int		false	"Limit (1-100, default 20)"
//	@Success		200		{object}	httpx.WebResponse[[]model.PublicPageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/pages [get]
func (ctl PublicController) ListPages(c fiber.Ctx) error {
	wid, err := strconv.ParseInt(c.Params("wid"), 10, 64)
	if err != nil || wid <= 0 {
		return httpx.WithCursor(c, []model.PublicPageResponse{}, "")
	}

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	limit := 20
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if l, err := strconv.Atoi(rawLimit); err == nil && l > 0 {
			limit = l
		}
	}

	var cursor *string
	if rawCursor := c.Query("cursor"); rawCursor != "" {
		cursor = &rawCursor
	}

	items, nextCursor, err := ctl.pageSvc.PublicList(ctx, wid, cursor, limit)
	if err != nil {
		return err
	}

	out := make([]model.PublicPageResponse, len(items))
	for i, it := range items {
		out[i] = toPublicPageResponse(it)
	}

	return httpx.WithCursor(c, out, nextCursor)
}

// GetPageBySlug retrieves a publicly visible page by slug.
//
//	@Summary		Get public page by slug
//	@Description	Retrieve a published page visible at current time by slug
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			slug	path		string	true	"Page slug"
//	@Success		200		{object}	httpx.WebResponse[model.PublicPageResponse]
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/pages/{slug} [get]
func (ctl PublicController) GetPageBySlug(c fiber.Ctx) error {
	wid, err := strconv.ParseInt(c.Params("wid"), 10, 64)
	if err != nil || wid <= 0 {
		return domain.ErrPageNotFound
	}
	slug := c.Params("slug")
	if slug == "" {
		return domain.ErrPageNotFound
	}

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	page, err := ctl.pageSvc.PublicGetBySlug(ctx, wid, slug)
	if err != nil {
		return domain.ErrPageNotFound
	}

	return httpx.OK(c, toPublicPageResponse(page))
}

func toPublicPostResponse(p domain.Post, apiURL string) model.PublicPostResponse {
	resp := model.PublicPostResponse{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Excerpt:     p.Excerpt,
		Body:        p.Body,
		PublishedAt: p.PublishAt,
	}
	if p.CoverMedia != nil {
		baseURL := strings.TrimRight(apiURL, "/")
		url := fmt.Sprintf("%s/public/workspaces/%d/cms/media/%d", baseURL, p.WorkspaceID, p.CoverMedia.ID)
		resp.CoverImage = &model.PublicCoverImage{
			ID:  p.CoverMedia.ID,
			Alt: p.CoverMedia.AltText,
			URL: url,
		}
	}
	return resp
}

func toPublicPageResponse(p domain.Page) model.PublicPageResponse {
	return model.PublicPageResponse{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Body:        p.Body,
		PublishedAt: p.PublishAt,
	}
}

// RedirectMedia redirects to the presigned download URL for a public media item.
//
//	@Summary		Redirect to public media
//	@Description	Redirects (302) to the presigned download URL for a media item that is public and not deleted
//	@Tags			CMS Public
//	@Param			wid	path	int	true	"Workspace ID"
//	@Param			id	path	int	true	"Media ID"
//	@Success		302	{string}	string	"Redirects to presigned URL"
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/media/{id} [get]
func (ctl PublicController) RedirectMedia(c fiber.Ctx) error {
	wid, err := strconv.ParseInt(c.Params("wid"), 10, 64)
	if err != nil || wid <= 0 {
		return domain.ErrMediaNotFound
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return domain.ErrMediaNotFound
	}

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	presignedURL, err := ctl.mediaSvc.GetPublicPresignedURL(ctx, wid, id)
	if err != nil {
		return domain.ErrMediaNotFound
	}

	return c.Redirect().Status(fiber.StatusFound).To(presignedURL)
}
