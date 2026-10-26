package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/internal/platform/s3x"
)

// FileServiceContract defines the file business logic operations required by FileController.
type FileServiceContract interface {
	Upload(c context.Context, req model.UploadFileRequest) (domain.File, error)
	GetByID(c context.Context, id int64) (domain.File, error)
	Search(c context.Context, req model.SearchFileRequest) ([]domain.File, string, error)
	Download(c context.Context, id int64) (domain.File, *s3x.Object, error)
	PresignDownload(c context.Context, id int64) (string, error)
	Update(c context.Context, id int64, req model.UpdateFileRequest) (domain.File, error)
	Delete(c context.Context, id int64) error
}

// FileController handles HTTP requests for workspace file uploads, downloads, search, and lifecycle.
type FileController struct {
	files FileServiceContract
}

// NewFileController constructs a new FileController backed by the specified FileServiceContract.
func NewFileController(files FileServiceContract) FileController {
	return FileController{files: files}
}

// Upload uploads a file into the workspace's S3 bucket and records its metadata.
//
//	@Summary		Upload workspace file
//	@Description	Upload a file via multipart/form-data into the workspace's dedicated S3 bucket
//	@Tags			Workspace Files
//	@Accept			mpfd
//	@Produce		json
//	@Param			wid			path		int		true	"Workspace ID"
//	@Param			file		formData	file	true	"File to upload"
//	@Param			name		formData	string	false	"Custom display filename"
//	@Param			entityType	formData	string	false	"Optional associated entity type (e.g. fin:transaction)"
//	@Param			entityId	formData	int		false	"Optional associated entity ID"
//	@Success		201			{object}	httpx.WebResponse[model.FileResponse]
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		422			{object}	httpx.ErrorResponse
//	@Failure		500			{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files [post]
func (ctl FileController) Upload(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}

	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return domain.ErrFileRequired
	}
	if fh.Size <= 0 {
		return domain.ErrFileEmpty
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	mimeType := strings.TrimSpace(fh.Header.Get("Content-Type"))
	if mimeType == "" || mimeType == "application/octet-stream" {
		buf := make([]byte, 512)
		n, _ := src.Read(buf)
		if n > 0 {
			mimeType = http.DetectContentType(buf[:n])
		}
		_, _ = src.Seek(0, io.SeekStart)
	}

	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		name = fh.Filename
	}

	entityType := strings.TrimSpace(c.FormValue("entityType"))
	var entityID int64
	if rawEntityID := strings.TrimSpace(c.FormValue("entityId")); rawEntityID != "" {
		parsedID, parseErr := strconv.ParseInt(rawEntityID, 10, 64)
		if parseErr != nil || parsedID <= 0 {
			return base.ErrInvalidID
		}
		entityID = parsedID
	}

	created, err := ctl.files.Upload(c, model.UploadFileRequest{
		WorkspaceID: wid,
		Name:        name,
		MimeType:    mimeType,
		Size:        fh.Size,
		EntityType:  entityType,
		EntityID:    entityID,
		Reader:      src,
	})
	if err != nil {
		return err
	}

	return httpx.Created(c, toFileResponse(created))
}

// Search searches files in a workspace with cursor pagination.
//
//	@Summary		Search workspace files
//	@Description	Search files in the workspace matching filter criteria with cursor pagination
//	@Tags			Workspace Files
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			request	body		model.SearchFileRequest	true	"Search filter criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.FileResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files/search [post]
func (ctl FileController) Search(c fiber.Ctx) error {
	return httpx.DoSearch(c, ctl.files.Search, toFileResponse)
}

// GetByID gets file metadata by ID.
//
//	@Summary		Get workspace file by ID
//	@Description	Retrieve file metadata by ID within a workspace
//	@Tags			Workspace Files
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"File ID"
//	@Success		200	{object}	httpx.WebResponse[model.FileResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files/{id} [get]
func (ctl FileController) GetByID(c fiber.Ctx) error {
	return httpx.DoGetByID(c, ctl.files.GetByID, toFileResponse)
}

// Download streams the binary content of a file from S3.
//
//	@Summary		Download workspace file
//	@Description	Download raw file content from the workspace's S3 bucket
//	@Tags			Workspace Files
//	@Produce		octet-stream
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"File ID"
//	@Success		200	{file}		binary
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files/{id}/download [get]
func (ctl FileController) Download(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	meta, obj, err := ctl.files.Download(c, id)
	if err != nil {
		return err
	}
	defer func() { _ = obj.Body.Close() }()

	data, err := io.ReadAll(obj.Body)
	if err != nil {
		return err
	}

	contentType := meta.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	safeName := strings.ReplaceAll(meta.Name, `"`, "")
	c.Set(fiber.HeaderContentType, contentType)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, safeName))
	return c.Status(fiber.StatusOK).Send(data)
}

// Presign generates a presigned GET URL for downloading the file directly from S3.
//
//	@Summary		Presign workspace file download URL
//	@Description	Generate a temporary presigned S3 GET URL for a file
//	@Tags			Workspace Files
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"File ID"
//	@Success		200	{object}	httpx.WebResponse[model.FileDownloadURLResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files/{id}/presign [get]
func (ctl FileController) Presign(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	urlStr, err := ctl.files.PresignDownload(c, id)
	if err != nil {
		return err
	}

	return httpx.OK(c, model.FileDownloadURLResponse{URL: urlStr})
}

// Update updates file metadata by ID.
//
//	@Summary		Update workspace file metadata
//	@Description	Update file display name or associated entity reference
//	@Tags			Workspace Files
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int						true	"Workspace ID"
//	@Param			id		path		int						true	"File ID"
//	@Param			request	body		model.UpdateFileRequest	true	"File update payload"
//	@Success		200		{object}	httpx.WebResponse[model.FileResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files/{id} [put]
func (ctl FileController) Update(c fiber.Ctx) error {
	return httpx.DoUpdate(c, ctl.files.Update, toFileResponse)
}

// Delete deletes a file from S3 and soft-deletes its metadata record.
//
//	@Summary		Delete workspace file
//	@Description	Delete the file object from S3 and soft-delete its metadata record in the database
//	@Tags			Workspace Files
//	@Produce		json
//	@Param			wid	path	int	true	"Workspace ID"
//	@Param			id	path	int	true	"File ID"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/files/{id} [delete]
func (ctl FileController) Delete(c fiber.Ctx) error {
	return httpx.DoDelete[int64](c, ctl.files.Delete)
}

func toFileResponse(f domain.File) model.FileResponse {
	return model.FileResponse{
		ID:          f.ID,
		WorkspaceID: f.WorkspaceID,
		Name:        f.Name,
		Bucket:      f.Bucket,
		Key:         f.Key,
		MimeType:    f.MimeType,
		Size:        f.Size,
		EntityType:  f.EntityType,
		EntityID:    f.EntityID,
		CreatedBy:   f.CreatedBy,
		CreatedAt:   f.CreatedAt,
		UpdatedBy:   f.UpdatedBy,
		UpdatedAt:   f.UpdatedAt,
	}
}
