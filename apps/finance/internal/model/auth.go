package model

import "github.com/google/uuid"

type AuthenticatedUser struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID uuid.UUID `json:"activeTeamId"`
	Roles       []string  `json:"role"`
}
