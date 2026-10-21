package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Post represents a CMS post domain entity.
type Post struct {
	ID              int64          `json:"id"`
	WorkspaceID     int64          `json:"workspaceId"`
	Title           string         `json:"title"`
	Slug            string         `json:"slug"`
	Excerpt         string         `json:"excerpt,omitempty"`
	Body            string         `json:"body"`
	CoverMediaID    *int64         `json:"coverMediaId,omitempty"`
	CoverMedia      *Media         `json:"coverImage,omitempty"`
	Status          string         `json:"status"`
	EffectiveStatus string         `json:"effectiveStatus"`
	PublishAt       *time.Time     `json:"publishAt,omitempty"`
	AuditMeta       base.AuditMeta `json:"auditMeta"`
}
