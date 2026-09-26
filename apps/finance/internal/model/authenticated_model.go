package model

import "github.com/google/uuid"

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
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Avatar        string          `json:"avatar"`
	Role          []UserRole      `json:"role"`
	WorkspaceID   uuid.UUID       `json:"activeTeamId"`
	WorkspaceName string          `json:"activeTeamName"`
	WorkspaceRole []WorkspaceRole `json:"activeOrganizationRole"`
}
