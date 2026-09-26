package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleAdmin      UserRole = "admin"
	UserRoleSuperadmin UserRole = "superadmin"
	UserRoleUser       UserRole = "user"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleAdmin  WorkspaceRole = "admin"
	WorkspaceRoleMember WorkspaceRole = "member"
)

type AuthenticatedUser struct {
	ID                 uuid.UUID       `json:"id"`
	Name               string          `json:"name"`
	Avatar             string          `json:"avatar,omitempty"`
	Role               []UserRole      `json:"role"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	WorkspaceID        uuid.UUID       `json:"activeTeamId"`
	WorkspaceName      string          `json:"activeTeamName"`
	WorkspaceRole      []WorkspaceRole `json:"activeOrganizationRole"`
	WorkspaceUpdatedAt time.Time       `json:"workspaceUpdatedAt"`
}
