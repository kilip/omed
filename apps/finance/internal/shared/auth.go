package shared

import (
	"context"

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
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	Avatar         string          `json:"avatar,omitempty"`
	WorkspaceID    uuid.UUID       `json:"activeWorkspaceId"`
	WorkspaceName  string          `json:"activeWorkspaceName"`
	WorkspaceRoles []WorkspaceRole `json:"activeWorkspaceRoles"`
}

const AUTH_USER_CONTEXT_KEY = "user"

func UserFromContext(ctx context.Context) AuthenticatedUser {
	u, _ := ctx.Value(AUTH_USER_CONTEXT_KEY).(AuthenticatedUser)
	return u
}
