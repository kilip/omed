package model

import (
	"time"
)

// PublicCoverImage represents cover image info in public post responses.
type PublicCoverImage struct {
	ID  int64  `json:"id"`
	Alt string `json:"alt"`
	URL string `json:"url"`
}

// PublicAlternate represents an alternate published translation for hreflang.
type PublicAlternate struct {
	Locale string `json:"locale"`
	Slug   string `json:"slug"`
	Path   string `json:"path"`
}

// PublicPostResponse represents publicly visible post details.
type PublicPostResponse struct {
	ID          int64             `json:"id"`
	Locale      string            `json:"locale"`
	Title       string            `json:"title"`
	Slug        string            `json:"slug"`
	Path        string            `json:"path"`
	Fallback    bool              `json:"fallback"`
	Excerpt     string            `json:"excerpt,omitempty"`
	Body        string            `json:"body"`
	PublishedAt *time.Time        `json:"publishedAt,omitempty"`
	CoverImage  *PublicCoverImage `json:"coverImage,omitempty"`
	Alternates  []PublicAlternate `json:"alternates,omitempty"`
}

// PublicPageResponse represents publicly visible page details.
type PublicPageResponse struct {
	ID          int64             `json:"id"`
	Locale      string            `json:"locale"`
	Title       string            `json:"title"`
	Slug        string            `json:"slug"`
	Path        string            `json:"path"`
	Fallback    bool              `json:"fallback"`
	Body        string            `json:"body"`
	PublishedAt *time.Time        `json:"publishedAt,omitempty"`
	Alternates  []PublicAlternate `json:"alternates,omitempty"`
}

// PublicListQuery defines cursor pagination parameters for public content listings.
type PublicListQuery struct {
	Cursor *string `query:"cursor"`
	Limit  *int    `query:"limit" validate:"omitempty,min=1,max=100"`
}
