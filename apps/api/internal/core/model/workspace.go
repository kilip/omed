// Package model provides Data Transfer Objects (DTOs) and request/response payloads for workspace endpoints.
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
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
	OwnerID     int64  `json:"ownerId,omitempty"`
}

// UpdateWorkspaceRequest defines the payload required to update an existing workspace.
type UpdateWorkspaceRequest struct {
	Name        string `json:"name,omitempty" validate:"required"`
	Description string `json:"description,omitempty"`
}

// SearchWorkspaceRequest defines filter criteria and cursor pagination options for searching workspaces.
type SearchWorkspaceRequest struct {
	Name    string `json:"name,omitempty"`
	OwnerID int64  `json:"ownerId,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}
