package model

import "github.com/google/uuid"

type AuthenticatedUser struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID uuid.UUID `json:"activeTeamId"`
	WorkspaceRoles []string `json:"activeOrganizationRole"`
	Roles       []string  `json:"role"`
}
