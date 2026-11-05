package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// TranslationSummary represents a summary of a translation for a post or page.
type TranslationSummary struct {
	Locale          string     `json:"locale"`
	Slug            string     `json:"slug"`
	Title           string     `json:"title"`
	Status          string     `json:"status"`
	EffectiveStatus string     `json:"effectiveStatus"`
	PublishAt       *time.Time `json:"publishAt,omitempty"`
}

// Post represents a CMS post domain entity.
type Post struct {
	ID              int64                `json:"id"`
	WorkspaceID     int64                `json:"workspaceId"`
	Locale          string               `json:"locale"`
	Title           string               `json:"title"`
	Slug            string               `json:"slug"`
	Excerpt         string               `json:"excerpt,omitempty"`
	Body            string               `json:"body"`
	CoverMediaID    *int64               `json:"coverMediaId,omitempty"`
	CoverMedia      *Media               `json:"coverImage,omitempty"`
	Status          string               `json:"status"`
	EffectiveStatus string               `json:"effectiveStatus"`
	PublishAt       *time.Time           `json:"publishAt,omitempty"`
	Translations    []TranslationSummary `json:"translations,omitempty"`
	AuditMeta       base.AuditMeta       `json:"auditMeta"`
}
