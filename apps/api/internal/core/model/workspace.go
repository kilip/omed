package model

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// WorkspaceResponse represents workspace metadata returned by workspace endpoints.
type WorkspaceResponse struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Owner       UserResponse `json:"owner"`
	CreatedBy   base.ByUser  `json:"createdBy"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedBy   base.ByUser  `json:"updatedBy"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// CreateWorkspaceRequest defines the payload required to create a new workspace.
type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
