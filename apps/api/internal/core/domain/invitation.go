package domain

import "time"

// Invitation represents a shareable workspace invitation link.
type Invitation struct {
	ID          int64
	WorkspaceID int64
	Token       string
	Roles       []string
	MaxUses     int
	UsesCount   int
	ExpiresAt   *time.Time
	IsRevoked   bool
	CreatedBy   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

