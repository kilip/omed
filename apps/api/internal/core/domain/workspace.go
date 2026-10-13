package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Workspace represents the domain entity for a workspace.
type Workspace struct {
	ID          int64
	Name        string
	Description string
	OwnerID     int64
	Owner       User
	Personal    bool
	CreatedByID *int64
	CreatedBy   base.ByUser
	UpdatedByID *int64
	UpdatedBy   base.ByUser
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
