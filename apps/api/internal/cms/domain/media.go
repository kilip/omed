package domain

import (
	"github.com/kilip/omed/api/internal/base"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
)

const (
	// MediaEntityType is the entityType stored on core_file records linked to CMS media.
	MediaEntityType = "cms:media"
)

// Media represents a CMS media item.
type Media struct {
	ID          int64            `json:"id"`
	WorkspaceID int64            `json:"workspaceId"`
	FileID      int64            `json:"fileId"`
	Title       string           `json:"title,omitempty"`
	AltText     string           `json:"altText,omitempty"`
	Caption     string           `json:"caption,omitempty"`
	IsPublic    bool             `json:"isPublic"`
	URL         string           `json:"url,omitempty"`
	File        *coredomain.File `json:"file,omitempty"`
	AuditMeta   base.AuditMeta   `json:"auditMeta"`
}
