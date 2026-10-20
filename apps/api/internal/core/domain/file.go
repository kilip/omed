package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// File represents the domain entity for a workspace file stored in S3.
type File struct {
	ID          int64
	WorkspaceID int64
	Name        string
	Bucket      string
	Key         string
	MimeType    string
	Size        int64
	EntityType  string
	EntityID    int64
	CreatedByID int64
	CreatedBy   base.ByUser
	UpdatedByID int64
	UpdatedBy   base.ByUser
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
