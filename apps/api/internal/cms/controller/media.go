package controller

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// MediaServiceContract defines the operations required by MediaController.
type MediaServiceContract interface {
	Upload(c context.Context, req model.UploadMediaRequest) (domain.Media, error)
	GetByID(c context.Context, id int64) (domain.Media, error)
	Search(c context.Context, req model.SearchMediaRequest) ([]domain.Media, string, error)
	Update(c context.Context, id int64, req model.UpdateMediaRequest) (domain.Media, error)
	Delete(c context.Context, id int64) error
}

// MediaController handles HTTP requests for CMS media management.
type MediaController struct {
	svc MediaServiceContract
}

// NewMediaController constructs a new MediaController backed by MediaServiceContract.
func NewMediaController(svc MediaServiceContract) MediaController {
	return MediaController{svc: svc}
}

// Upload uploads a new media file into the workspace.
//
//	@Summary		Upload media
//	@Description	Upload a file via multipart/form-data as CMS media
//	@Tags			CMS Media
//	@Accept			mpfd
//	@Produce		json
//	@Param			wid			path		int		true	"Workspace ID"
//	@Param			file		formData	file	true	"Media file to upload"
//	@Param			title		formData	string	false	"Media title"
//	@Param			altText		formData	string	false	"Media alt text"
//	@Param			caption		formData	string	false	"Media caption"
//	@Param			isPublic	formData	bool	false	"Is media public (defaults to true)"
//	@Success		201			{object}	httpx.WebResponse[model.MediaResponse]
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		422			{object}	httpx.ErrorResponse
//	@Failure		500			{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/media [post]
func (ctl MediaController) Upload(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}

	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return coredomain.ErrFileRequired
	}
	if fh.Size <= 0 {
		return coredomain.ErrFileEmpty
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	mimeType := strings.TrimSpace(fh.Header.Get("Content-Type"))
	if mimeType == "" || mimeType == "application/octet-stream" {
		buf := make([]byte, 512)
		n, _ := src.Read(buf)
		if n > 0 {
			mimeType = http.DetectContentType(buf[:n])
		}
		_, _ = src.Seek(0, io.SeekStart)
	}

	name := fh.Filename

	title := strings.TrimSpace(c.FormValue("title"))
	altText := strings.TrimSpace(c.FormValue("altText"))
	caption := strings.TrimSpace(c.FormValue("caption"))

	var isPublic *bool
	if rawPublic := strings.TrimSpace(c.FormValue("isPublic")); rawPublic != "" {
		val := rawPublic == "true" || rawPublic == "1"
		isPublic = &val
	}

	created, err := ctl.svc.Upload(c, model.UploadMediaRequest{
		WorkspaceID: wid,
		Title:       title,
		AltText:     altText,
		Caption:     caption,
		IsPublic:    isPublic,
		Name:        name,
		MimeType:    mimeType,
		Size:        fh.Size,
		Reader:      src,
	})
	if err != nil {
		return err
	}

	return httpx.Created(c, toMediaResponse(created))
}

// Search searches media in the workspace with query and filters.
//
//	@Summary		Search media
//	@Description	Search media in the workspace with filters and cursor pagination
//	@Tags			CMS Media
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SearchMediaRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.MediaResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/media/search [post]
func (ctl MediaController) Search(c fiber.Ctx) error {
	var req model.SearchMediaRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.svc.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.MediaResponse, len(items))
	for i, it := range items {
		out[i] = toMediaResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// GetByID retrieves a media record by ID.
//
//	@Summary		Get media by ID
//	@Description	Get media metadata and presigned URL by ID
//	@Tags			CMS Media
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Media ID"
//	@Success		200	{object}	httpx.WebResponse[model.MediaResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/media/{id} [get]
func (ctl MediaController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	item, err := ctl.svc.GetByID(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toMediaResponse(item))
}

// Update updates media metadata by ID.
//
//	@Summary		Update media
//	@Description	Update media metadata (title, alt text, caption, isPublic)
//	@Tags			CMS Media
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			id		path		int						true	"Media ID"
//	@Param			request	body		model.UpdateMediaRequest	true	"Update payload"
//	@Success		200		{object}	httpx.WebResponse[model.MediaResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/media/{id} [put]
func (ctl MediaController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateMediaRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	item, err := ctl.svc.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toMediaResponse(item))
}

// Delete soft-deletes a media item and removes the underlying file.
//
//	@Summary		Delete media
//	@Description	Delete media item and underlying file (fails if media is referenced by a post)
//	@Tags			CMS Media
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Media ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/cms/media/{id} [delete]
func (ctl MediaController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.svc.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

func toMediaResponse(m domain.Media) model.MediaResponse {
	resp := model.MediaResponse{
		ID:        m.ID,
		Title:     m.Title,
		AltText:   m.AltText,
		Caption:   m.Caption,
		IsPublic:  m.IsPublic,
		URL:       m.URL,
		AuditMeta: m.AuditMeta,
	}
	if m.File != nil {
		resp.Name = m.File.Name
		resp.MimeType = m.File.MimeType
		resp.Size = m.File.Size
		resp.File = &coremodel.FileResponse{
			ID:          m.File.ID,
			WorkspaceID: m.File.WorkspaceID,
			Name:        m.File.Name,
			Bucket:      m.File.Bucket,
			Key:         m.File.Key,
			MimeType:    m.File.MimeType,
			Size:        m.File.Size,
			EntityType:  m.File.EntityType,
			EntityID:    m.File.EntityID,
			CreatedBy:   m.File.CreatedBy,
			CreatedAt:   m.File.CreatedAt,
			UpdatedBy:   m.File.UpdatedBy,
			UpdatedAt:   m.File.UpdatedAt,
		}
	}
	return resp
}
