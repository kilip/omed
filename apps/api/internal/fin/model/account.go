// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// AccountResponse represents Chart of Accounts entity metadata returned by API endpoints.
type AccountResponse struct {
	ID            int64        `json:"id"`
	WorkspaceID   int64        `json:"workspaceId"`
	Code          string       `json:"code"`
	Name          string       `json:"name"`
	Type          string       `json:"type"`
	NormalBalance string       `json:"normalBalance"`
	ParentID      *int64       `json:"parentId,omitempty"`
	Description   string       `json:"description,omitempty"`
	IsActive      bool         `json:"isActive"`
	IsPostable    bool         `json:"isPostable"`
	DeletedAt     *time.Time   `json:"deletedAt,omitempty"`
	DeletedBy     *base.ByUser `json:"deletedBy,omitempty"`
	CreatedBy     base.ByUser  `json:"createdBy"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedBy     base.ByUser  `json:"updatedBy"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

// AccountTreeNodeResponse represents an account in a hierarchical tree view.
type AccountTreeNodeResponse struct {
	ID            int64                     `json:"id"`
	WorkspaceID   int64                     `json:"workspaceId"`
	Code          string                    `json:"code"`
	Name          string                    `json:"name"`
	Type          string                    `json:"type"`
	NormalBalance string                    `json:"normalBalance"`
	ParentID      *int64                    `json:"parentId,omitempty"`
	Description   string                    `json:"description,omitempty"`
	IsActive      bool                      `json:"isActive"`
	IsPostable    bool                      `json:"isPostable"`
	Children      []AccountTreeNodeResponse `json:"children"`
	CreatedBy     base.ByUser               `json:"createdBy"`
	CreatedAt     time.Time                 `json:"createdAt"`
	UpdatedBy     base.ByUser               `json:"updatedBy"`
	UpdatedAt     time.Time                 `json:"updatedAt"`
}

// CreateAccountRequest defines the payload required to create a new chart of accounts entry.
type CreateAccountRequest struct {
	Code          string `json:"code" validate:"required"`
	Name          string `json:"name" validate:"required"`
	Type          string `json:"type" validate:"required,oneof=ASSET LIABILITY EQUITY REVENUE EXPENSE"`
	NormalBalance string `json:"normalBalance,omitempty" validate:"omitempty,oneof=DEBIT CREDIT"`
	ParentID      *int64 `json:"parentId,omitempty"`
	Description   string `json:"description,omitempty"`
	IsActive      *bool  `json:"isActive,omitempty"`
	IsPostable    *bool  `json:"isPostable,omitempty"`
}

// UpdateAccountRequest defines the payload required to update an existing account.
type UpdateAccountRequest struct {
	Name          string `json:"name" validate:"required"`
	Code          string `json:"code,omitempty"`
	NormalBalance string `json:"normalBalance,omitempty" validate:"omitempty,oneof=DEBIT CREDIT"`
	ParentID      *int64 `json:"parentId,omitempty"`
	Description   string `json:"description,omitempty"`
	IsActive      *bool  `json:"isActive,omitempty"`
	IsPostable    *bool  `json:"isPostable,omitempty"`
}

// SearchAccountRequest defines filter criteria and pagination for accounts search.
type SearchAccountRequest struct {
	Query          string `json:"query,omitempty"`
	Code           string `json:"code,omitempty"`
	Type           string `json:"type,omitempty"`
	ParentID       *int64 `json:"parentId,omitempty"`
	IsActive       *bool  `json:"isActive,omitempty"`
	IsPostable     *bool  `json:"isPostable,omitempty"`
	IncludeDeleted bool   `json:"includeDeleted,omitempty"`
	Cursor         string `json:"cursor,omitempty"`
	Limit          int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}
