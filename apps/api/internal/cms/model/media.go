package model

import (
	"io"

	"github.com/kilip/omed/api/internal/base"
	coremodel "github.com/kilip/omed/api/internal/core/model"
)

// UploadMediaRequest defines the parameters for uploading a new media file.
type UploadMediaRequest struct {
	WorkspaceID int64
	Title       string
	AltText     string
	Caption     string
	IsPublic    *bool
	Name        string
	MimeType    string
	Size        int64
	Reader      io.Reader
}

// CreateMediaRecordRequest defines internal payload for inserting a cms_media database record.
type CreateMediaRecordRequest struct {
	WorkspaceID int64
	FileID      int64
	Title       string
	AltText     string
	Caption     string
	IsPublic    bool
}

// UpdateMediaRequest defines the metadata fields that can be updated on a media record.
type UpdateMediaRequest struct {
	Title    *string `json:"title,omitempty"`
	AltText  *string `json:"altText,omitempty"`
	Caption  *string `json:"caption,omitempty"`
	IsPublic *bool   `json:"isPublic,omitempty"`
}

// SearchMediaRequest defines filtering and pagination parameters for media search.
type SearchMediaRequest struct {
	Query    *string `json:"query,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`
	Cursor   *string `json:"cursor,omitempty"`
	Limit    *int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// MediaResponse represents media item metadata returned by API endpoints.
type MediaResponse struct {
	ID        int64                   `json:"id"`
	Title     string                  `json:"title,omitempty"`
	AltText   string                  `json:"altText,omitempty"`
	Caption   string                  `json:"caption,omitempty"`
	IsPublic  bool                    `json:"isPublic"`
	Name      string                  `json:"name"`
	MimeType  string                  `json:"mimeType"`
	Size      int64                   `json:"size"`
	URL       string                  `json:"url"`
	File      *coremodel.FileResponse `json:"file,omitempty"`
	AuditMeta base.AuditMeta          `json:"auditMeta"`
}
