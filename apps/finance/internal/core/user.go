package core

import (
	"context"
	"slices"
	"strings"

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
	Scope          string          `json:"scope,omitempty"`
}

func (u AuthenticatedUser) HasScope(requiredScope string) bool {
	if u.Scope == "" || requiredScope == "" {
		return true
	}
	fields := strings.Fields(u.Scope)
	return slices.Contains(fields, requiredScope)
}

const AUTH_USER_CONTEXT_KEY = "user"

func UserFromContext(ctx context.Context) AuthenticatedUser {
	u, _ := ctx.Value(AUTH_USER_CONTEXT_KEY).(AuthenticatedUser)
	return u
}

func ContextWithUser(ctx context.Context, u AuthenticatedUser) context.Context {
	return context.WithValue(ctx, AUTH_USER_CONTEXT_KEY, u)
}
