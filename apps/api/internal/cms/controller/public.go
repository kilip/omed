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
	PublicList(c context.Context, workspaceID int64, locale string, cursor *string, limit int) ([]domain.Post, string, error)
	PublicGetBySlug(c context.Context, workspaceID int64, locale, slug string) (domain.Post, bool, []model.PublicAlternate, error)
}

// PublicPageServiceContract defines public page retrieval operations.
type PublicPageServiceContract interface {
	PublicList(c context.Context, workspaceID int64, locale string, cursor *string, limit int) ([]domain.Page, string, error)
	PublicGetBySlug(c context.Context, workspaceID int64, locale, slug string) (domain.Page, bool, []model.PublicAlternate, error)
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

// ListPosts lists publicly visible posts for a workspace in default locale.
//
//	@Summary		List public posts
//	@Description	List published posts visible at the current time for a workspace in default locale
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

	locale := c.Params("locale")

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

	items, nextCursor, err := ctl.postSvc.PublicList(ctx, wid, locale, cursor, limit)
	if err != nil {
		return err
	}

	out := make([]model.PublicPostResponse, len(items))
	for i, it := range items {
		out[i] = toPublicPostResponse(it, false, nil, ctl.apiURL)
	}

	return httpx.WithCursor(c, out, nextCursor)
}

// ListPostsWithLocale lists publicly visible posts for a workspace in specified locale.
//
//	@Summary		List public posts with locale
//	@Description	List published posts visible at current time for a workspace in specified locale
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			locale	path		string	true	"BCP 47 locale tag"
//	@Param			cursor	query		string	false	"Composite pagination cursor"
//	@Param			limit	query		int		false	"Limit (1-100, default 20)"
//	@Success		200		{object}	httpx.WebResponse[[]model.PublicPostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/{locale}/posts [get]
func (ctl PublicController) ListPostsWithLocale(c fiber.Ctx) error {
	return ctl.ListPosts(c)
}

// GetPostBySlug retrieves a publicly visible post by slug in default locale.
//
//	@Summary		Get public post by slug
//	@Description	Retrieve a published post visible at current time by slug in default locale
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			slug	path		string	true	"Post slug"
//	@Success		200		{object}	httpx.WebResponse[model.PublicPostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
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
	locale := c.Params("locale")

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	post, fallback, alternates, err := ctl.postSvc.PublicGetBySlug(ctx, wid, locale, slug)
	if err != nil {
		return err
	}

	return httpx.OK(c, toPublicPostResponse(post, fallback, alternates, ctl.apiURL))
}

// GetPostBySlugWithLocale retrieves a publicly visible post by slug in specified locale.
//
//	@Summary		Get public post by slug with locale
//	@Description	Retrieve a published post visible at current time by slug in specified locale
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			locale	path		string	true	"BCP 47 locale tag"
//	@Param			slug	path		string	true	"Post slug"
//	@Success		200		{object}	httpx.WebResponse[model.PublicPostResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/{locale}/posts/{slug} [get]
func (ctl PublicController) GetPostBySlugWithLocale(c fiber.Ctx) error {
	return ctl.GetPostBySlug(c)
}

// ListPages lists publicly visible pages for a workspace in default locale.
//
//	@Summary		List public pages
//	@Description	List published pages visible at the current time for a workspace in default locale
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

	locale := c.Params("locale")

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

	items, nextCursor, err := ctl.pageSvc.PublicList(ctx, wid, locale, cursor, limit)
	if err != nil {
		return err
	}

	out := make([]model.PublicPageResponse, len(items))
	for i, it := range items {
		out[i] = toPublicPageResponse(it, false, nil)
	}

	return httpx.WithCursor(c, out, nextCursor)
}

// ListPagesWithLocale lists publicly visible pages for a workspace in specified locale.
//
//	@Summary		List public pages with locale
//	@Description	List published pages visible at current time for a workspace in specified locale
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			locale	path		string	true	"BCP 47 locale tag"
//	@Param			cursor	query		string	false	"Composite pagination cursor"
//	@Param			limit	query		int		false	"Limit (1-100, default 20)"
//	@Success		200		{object}	httpx.WebResponse[[]model.PublicPageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/{locale}/pages [get]
func (ctl PublicController) ListPagesWithLocale(c fiber.Ctx) error {
	return ctl.ListPages(c)
}

// GetPageBySlug retrieves a publicly visible page by slug in default locale.
//
//	@Summary		Get public page by slug
//	@Description	Retrieve a published page visible at current time by slug in default locale
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			slug	path		string	true	"Page slug"
//	@Success		200		{object}	httpx.WebResponse[model.PublicPageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
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
	locale := c.Params("locale")

	ctx := base.WithWorkspaceID(c.Context(), wid)
	c.SetContext(ctx)

	page, fallback, alternates, err := ctl.pageSvc.PublicGetBySlug(ctx, wid, locale, slug)
	if err != nil {
		return err
	}

	return httpx.OK(c, toPublicPageResponse(page, fallback, alternates))
}

// GetPageBySlugWithLocale retrieves a publicly visible page by slug in specified locale.
//
//	@Summary		Get public page by slug with locale
//	@Description	Retrieve a published page visible at current time by slug in specified locale
//	@Tags			CMS Public
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			locale	path		string	true	"BCP 47 locale tag"
//	@Param			slug	path		string	true	"Page slug"
//	@Success		200		{object}	httpx.WebResponse[model.PublicPageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/public/workspaces/{wid}/cms/{locale}/pages/{slug} [get]
func (ctl PublicController) GetPageBySlugWithLocale(c fiber.Ctx) error {
	return ctl.GetPageBySlug(c)
}

func toPublicPostResponse(p domain.Post, fallback bool, alternates []model.PublicAlternate, apiURL string) model.PublicPostResponse {
	resp := model.PublicPostResponse{
		ID:          p.ID,
		Locale:      p.Locale,
		Title:       p.Title,
		Slug:        p.Slug,
		Path:        fmt.Sprintf("/%s/%s", p.Locale, p.Slug),
		Fallback:    fallback,
		Excerpt:     p.Excerpt,
		Body:        p.Body,
		PublishedAt: p.PublishAt,
		Alternates:  alternates,
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

func toPublicPageResponse(p domain.Page, fallback bool, alternates []model.PublicAlternate) model.PublicPageResponse {
	return model.PublicPageResponse{
		ID:          p.ID,
		Locale:      p.Locale,
		Title:       p.Title,
		Slug:        p.Slug,
		Path:        fmt.Sprintf("/%s/%s", p.Locale, p.Slug),
		Fallback:    fallback,
		Body:        p.Body,
		PublishedAt: p.PublishAt,
		Alternates:  alternates,
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
