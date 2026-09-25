package model

import "github.com/google/uuid"

type AuthenticatedUser struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	WorkspaceID    uuid.UUID `json:"activeTeamId"`
	WorkspaceName  string    `json:"activeTeamName"`
	WorkspaceRoles []string  `json:"activeOrganizationRole"`
	Roles          []string  `json:"role"`
	Avatar         string    `json:"avatar"`
}
