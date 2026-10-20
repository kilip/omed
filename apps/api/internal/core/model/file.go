package model

import (
	"io"
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// FileResponse represents file metadata returned by file endpoints.
type FileResponse struct {
	ID          int64       `json:"id"`
	WorkspaceID int64       `json:"workspaceId"`
	Name        string      `json:"name"`
	Bucket      string      `json:"bucket"`
	Key         string      `json:"key"`
	MimeType    string      `json:"mimeType"`
	Size        int64       `json:"size"`
	EntityType  string      `json:"entityType,omitempty"`
	EntityID    int64       `json:"entityId,omitempty"`
	CreatedBy   base.ByUser `json:"createdBy"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedBy   base.ByUser `json:"updatedBy"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// FileDownloadURLResponse represents a presigned download URL response.
type FileDownloadURLResponse struct {
	URL string `json:"url"`
}

// UploadFileRequest defines the input required by FileService to upload a file to S3 and persist metadata.
type UploadFileRequest struct {
	WorkspaceID int64
	Name        string
	MimeType    string
	Size        int64
	EntityType  string
	EntityID    int64
	Reader      io.Reader
}

// CreateFileRecordRequest defines the persistence input for creating a file metadata record.
type CreateFileRecordRequest struct {
	WorkspaceID int64
	Name        string
	Bucket      string
	Key         string
	MimeType    string
	Size        int64
	EntityType  string
	EntityID    int64
}

// UpdateFileRequest defines the payload required to update file metadata.
type UpdateFileRequest struct {
	Name       string `json:"name" validate:"required"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   int64  `json:"entityId,omitempty"`
}

// SearchFileRequest defines filter criteria and cursor pagination options for searching files.
type SearchFileRequest struct {
	Name       string `json:"name,omitempty"`
	MimeType   string `json:"mimeType,omitempty"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   int64  `json:"entityId,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}
