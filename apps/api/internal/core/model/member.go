package model

import "time"

// MemberResponse represents a workspace member returned by member endpoints.
type MemberResponse struct {
	ID          int64        `json:"id"`
	WorkspaceID int64        `json:"workspaceId"`
	User        UserResponse `json:"user"`
	Roles       []string     `json:"roles"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// InviteMemberRequest defines the payload to add a user (by email) to a workspace.
// If no user with the email exists, a new user is created.
type InviteMemberRequest struct {
	Email string   `json:"email" validate:"required,email"`
	Roles []string `json:"roles" validate:"required,min=1,dive,required"`
}

// UpdateMemberRequest defines the payload to replace a member's roles.
type UpdateMemberRequest struct {
	Roles []string `json:"roles" validate:"required,min=1,dive,required"`
}

// SearchMemberRequest defines cursor pagination options for listing workspace members.
type SearchMemberRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// PermissionItem is a single (object, action) permission.
type PermissionItem struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

// PermissionsResponse describes the effective authorization of the current user in a workspace.
type PermissionsResponse struct {
	WorkspaceID int64            `json:"workspaceId"`
	Owner       bool             `json:"owner"`
	Roles       []string         `json:"roles"`
	Permissions []PermissionItem `json:"permissions"`
}

