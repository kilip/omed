package domain

import "time"

// Member represents a user's membership in a workspace.
type Member struct {
	ID          int64
	WorkspaceID int64
	User        User
	Roles       []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
