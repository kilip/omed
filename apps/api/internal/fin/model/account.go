// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"github.com/kilip/omed/api/internal/base"
)

// AccountResponse represents Chart of Accounts entity metadata returned by API endpoints.
type AccountResponse struct {
	ID            int64          `json:"id"`
	WorkspaceID   int64          `json:"workspaceId"`
	Code          string         `json:"code"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	NormalBalance string         `json:"normalBalance"`
	Currency      string         `json:"currency,omitempty"`
	ParentID      int64          `json:"parentId,omitempty"`
	Description   string         `json:"description,omitempty"`
	IsActive      bool           `json:"isActive"`
	IsPostable    bool           `json:"isPostable"`
	AuditMeta     base.AuditMeta `json:"auditMeta"`
}

// CreateAccountRequest defines the payload required to create a new chart of accounts entry.
type CreateAccountRequest struct {
	Code          string `json:"code" validate:"required"`
	Name          string `json:"name" validate:"required"`
	Type          string `json:"type" validate:"required,oneof=ASSET LIABILITY EQUITY REVENUE EXPENSE"`
	NormalBalance string `json:"normalBalance,omitempty" validate:"omitempty,oneof=DEBIT CREDIT"`
	Currency      string `json:"currency,omitempty" validate:"omitempty,len=3"`
	ParentID      int64  `json:"parentId,omitempty"`
	Description   string `json:"description,omitempty"`
	IsActive      *bool  `json:"isActive,omitempty"`
	IsPostable    *bool  `json:"isPostable,omitempty"`
}

// UpdateAccountRequest defines the payload required to update an existing account.
type UpdateAccountRequest struct {
	Name          string `json:"name" validate:"required"`
	Code          string `json:"code,omitempty"`
	NormalBalance string `json:"normalBalance,omitempty" validate:"omitempty,oneof=DEBIT CREDIT"`
	ParentID      int64  `json:"parentId,omitempty"`
	Description   string `json:"description,omitempty"`
	IsActive      *bool  `json:"isActive,omitempty"`
	IsPostable    *bool  `json:"isPostable,omitempty"`
}

// SearchAccountRequest defines request payload for searching accounts in a workspace.
type SearchAccountRequest struct {
}

// SeedAccountsRequest defines the payload for seeding a chart of accounts template.
type SeedAccountsRequest struct {
	Template string `json:"template" validate:"required"`
	Lang     string `json:"lang" validate:"required"`
}

// SeedAccountsResponse represents the outcome of seeding accounts.
type SeedAccountsResponse struct {
	Count int `json:"count"`
}

// AccountTemplateItemResponse represents a single account record in a template preview.
type AccountTemplateItemResponse struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	NormalBalance string  `json:"normalBalance"`
	ParentCode    *string `json:"parentCode,omitempty"`
	Description   string  `json:"description,omitempty"`
	IsPostable    bool    `json:"isPostable"`
}

// AccountTemplatePreviewResponse represents the preview payload for a COA template.
type AccountTemplatePreviewResponse struct {
	Template string                        `json:"template"`
	Lang     string                        `json:"lang"`
	Accounts []AccountTemplateItemResponse `json:"accounts"`
}
