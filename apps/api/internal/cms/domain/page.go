package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Page represents a CMS page domain entity.
type Page struct {
	ID              int64          `json:"id"`
	WorkspaceID     int64          `json:"workspaceId"`
	Title           string         `json:"title"`
	Slug            string         `json:"slug"`
	Body            string         `json:"body"`
	Status          string         `json:"status"`
	EffectiveStatus string         `json:"effectiveStatus"`
	PublishAt       *time.Time     `json:"publishAt,omitempty"`
	AuditMeta       base.AuditMeta `json:"auditMeta"`
}
