package model

import (
	"time"

	"github.com/kilip/omed/api/internal/core"
)

type WorkspaceResponse struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Owner       UserResponse `json:"owner"`
	CreatedBy   core.ByUser  `json:"createdBy"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedBy   core.ByUser  `json:"updatedBy"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
