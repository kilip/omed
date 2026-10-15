package model

import "time"

// CreateInvitationRequest defines the payload to create a workspace invitation link.
type CreateInvitationRequest struct {
	Roles     []string   `json:"roles" validate:"required,min=1,dive,required"`
	MaxUses   int        `json:"maxUses,omitempty" validate:"omitempty,min=0"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Email     string     `json:"email,omitempty" validate:"omitempty,email"`
}

// SendInvitationRequest defines the payload to send or resend an invitation email.
type SendInvitationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendInvitationResponse defines the response returned after sending an invitation email.
type SendInvitationResponse struct {
	Sent bool `json:"sent"`
}

// SearchInvitationRequest defines cursor pagination options for listing workspace invitations.
type SearchInvitationRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// InvitationResponse represents an invitation returned by management endpoints.
type InvitationResponse struct {
	ID          int64      `json:"id"`
	WorkspaceID int64      `json:"workspaceId"`
	Token       string     `json:"token"`
	Roles       []string   `json:"roles"`
	MaxUses     int        `json:"maxUses"`
	UsesCount   int        `json:"usesCount"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	IsRevoked   bool       `json:"isRevoked"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// PublicInvitationResponse provides preview information for users opening an invitation link.
type PublicInvitationResponse struct {
	WorkspaceID          int64      `json:"workspaceId"`
	WorkspaceName        string     `json:"workspaceName"`
	WorkspaceDescription string     `json:"workspaceDescription,omitempty"`
	InviterName          string     `json:"inviterName"`
	Roles                []string   `json:"roles"`
	ExpiresAt            *time.Time `json:"expiresAt,omitempty"`
	IsValid              bool       `json:"isValid"`
}
