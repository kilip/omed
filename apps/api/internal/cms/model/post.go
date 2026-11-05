package model

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
)

// CreatePostRequest defines the request body for creating a post.
type CreatePostRequest struct {
	Title        string  `json:"title" validate:"required"`
	Slug         *string `json:"slug,omitempty"`
	Excerpt      *string `json:"excerpt,omitempty"`
	Body         *string `json:"body,omitempty"`
	CoverMediaID *int64  `json:"coverMediaId,omitempty"`
	Locale       *string `json:"locale,omitempty"`
}

// UpdatePostRequest defines the request body for updating a post.
type UpdatePostRequest struct {
	Title        *string `json:"title,omitempty"`
	Slug         *string `json:"slug,omitempty"`
	Excerpt      *string `json:"excerpt,omitempty"`
	Body         *string `json:"body,omitempty"`
	CoverMediaID *int64  `json:"coverMediaId,omitempty"`
	Locale       *string `json:"locale,omitempty"`
}

// SavePostTranslationRequest defines payload for creating or updating a post translation.
type SavePostTranslationRequest struct {
	Title   string  `json:"title" validate:"required"`
	Slug    *string `json:"slug,omitempty"`
	Excerpt *string `json:"excerpt,omitempty"`
	Body    *string `json:"body,omitempty"`
}

// SchedulePostRequest defines the request body for scheduling post publication.
type SchedulePostRequest struct {
	PublishAt time.Time `json:"publishAt" validate:"required"`
}

// SearchPostRequest defines filtering and pagination parameters for searching posts.
type SearchPostRequest struct {
	Query  *string `json:"query,omitempty"`
	Status *string `json:"status,omitempty"` // Filter on effective status: DRAFT, SCHEDULED, PUBLISHED
	Locale *string `json:"locale,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
	Limit  *int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// PostResponse represents post metadata returned by admin API endpoints.
type PostResponse struct {
	ID              int64                     `json:"id"`
	WorkspaceID     int64                     `json:"workspaceId"`
	Locale          string                    `json:"locale"`
	Title           string                    `json:"title"`
	Slug            string                    `json:"slug"`
	Excerpt         string                    `json:"excerpt,omitempty"`
	Body            string                    `json:"body"`
	CoverMediaID    *int64                    `json:"coverMediaId,omitempty"`
	CoverImage      *MediaResponse            `json:"coverImage,omitempty"`
	Status          string                    `json:"status"`
	EffectiveStatus string                    `json:"effectiveStatus"`
	PublishAt       *time.Time                `json:"publishAt,omitempty"`
	Translations    []domain.TranslationSummary `json:"translations,omitempty"`
	AuditMeta       base.AuditMeta            `json:"auditMeta"`
}

// CreatePostRecord defines internal payload for inserting a cms_post and initial translation record.
type CreatePostRecord struct {
	WorkspaceID  int64
	Locale       string
	Title        string
	Slug         string
	Excerpt      string
	Body         string
	CoverMediaID *int64
	Status       string
	PublishAt    *time.Time
}

// UpdatePostRecord defines internal payload for updating a cms_post and translation record.
type UpdatePostRecord struct {
	Locale          string
	Title           *string
	Slug            *string
	Excerpt         *string
	Body            *string
	CoverMediaID    *int64
	ClearCoverMedia bool
	Status          *string
	PublishAt       *time.Time
	ClearPublishAt  bool
}

// SavePostTranslationRecord defines internal payload for upserting a cms_post_translation record.
type SavePostTranslationRecord struct {
	PostID      int64
	WorkspaceID int64
	Locale      string
	Title       string
	Slug        string
	Excerpt     string
	Body        string
	Status      string
	PublishAt   *time.Time
}
