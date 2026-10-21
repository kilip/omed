package model

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// CreatePageRequest defines the request body for creating a page.
type CreatePageRequest struct {
	Title string  `json:"title" validate:"required"`
	Slug  *string `json:"slug,omitempty"`
	Body  *string `json:"body,omitempty"`
}

// UpdatePageRequest defines the request body for updating a page.
type UpdatePageRequest struct {
	Title *string `json:"title,omitempty"`
	Slug  *string `json:"slug,omitempty"`
	Body  *string `json:"body,omitempty"`
}

// SchedulePageRequest defines the request body for scheduling page publication.
type SchedulePageRequest struct {
	PublishAt time.Time `json:"publishAt" validate:"required"`
}

// SearchPageRequest defines filtering and pagination parameters for searching pages.
type SearchPageRequest struct {
	Query  *string `json:"query,omitempty"`
	Status *string `json:"status,omitempty"` // Filter on effective status: DRAFT, SCHEDULED, PUBLISHED
	Cursor *string `json:"cursor,omitempty"`
	Limit  *int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// PageResponse represents page metadata returned by admin API endpoints.
type PageResponse struct {
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

// CreatePageRecord defines internal payload for inserting a cms_page database record.
type CreatePageRecord struct {
	WorkspaceID int64
	Title       string
	Slug        string
	Body        string
	Status      string
	PublishAt   *time.Time
}

// UpdatePageRecord defines internal payload for updating a cms_page database record.
type UpdatePageRecord struct {
	Title          *string
	Slug           *string
	Body           *string
	Status         *string
	PublishAt      *time.Time
	ClearPublishAt bool
}
