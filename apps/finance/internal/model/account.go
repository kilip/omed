package model

import (
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeRevenue   AccountType = "revenue"
	AccountTypeExpense   AccountType = "expense"
)

type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusArchived AccountStatus = "archived"
)

// Account is the response DTO for ent Account (chart of accounts entry).
type AccountResponse struct {
	ID            uuid.UUID     `json:"id"`
	WorkspaceID   uuid.UUID     `json:"workspaceId"`
	Code          string        `json:"code"`
	Name          string        `json:"name"`
	Description   *string       `json:"description,omitempty"`
	Type          AccountType   `json:"type" enums:"asset,liability,equity,revenue,expense"`
	Currency      string        `json:"currency"`
	Status        AccountStatus `json:"status" enums:"active,archived"`
	ParentID      *uuid.UUID    `json:"parentId,omitempty"`
	CreatedBy     uuid.UUID     `json:"createdBy"`
	CreatedByName string        `json:"createdByName"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedBy     uuid.UUID     `json:"updatedBy"`
	UpdatedByName string        `json:"updatedByName"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type CreateAccountRequest struct {
	Code        string      `json:"code" validate:"required,max=32"`
	Name        string      `json:"name" validate:"required,max=255"`
	Description *string     `json:"description,omitempty" validate:"omitempty,max=1000"`
	Type        AccountType `json:"type" validate:"required,oneof=asset liability equity revenue expense" enums:"asset,liability,equity,revenue,expense"`
	Currency    string      `json:"currency" validate:"required,len=3,uppercase" example:"IDR"`
	ParentID    *uuid.UUID  `json:"parentId,omitempty"`
}

// UpdateAccountRequest: code, type & currency sengaja tidak bisa diubah.
type UpdateAccountRequest struct {
	Name        *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string        `json:"description,omitempty" validate:"omitempty,max=1000"`
	Status      *AccountStatus `json:"status,omitempty" validate:"omitempty,oneof=active archived" enums:"active,archived"`
	ParentID    *uuid.UUID     `json:"parentId,omitempty"`
}

type SeedAccountRequest struct {
	Profile  string `json:"profile" validate:"required"`
	Lang     string `json:"lang" validate:"required"`
	Currency string `json:"currency" validate:"omitempty,len=3,uppercase" example:"IDR"`
	Force    bool   `json:"force"`
}

type ListAccountRequest struct {
	Type   AccountType   `query:"type" validate:"omitempty,oneof=asset liability equity revenue expense"`
	Status AccountStatus `query:"status" validate:"omitempty,oneof=active archived"`
}

type SeedTemplateResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
}

type SeedPreviewAccount struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Type       string  `json:"type" enums:"asset,liability,equity,revenue,expense"`
	ParentCode *string `json:"parentCode,omitempty"`
}

type SeedPreviewResponse struct {
	Profile      string               `json:"profile"`
	Lang         string               `json:"lang"`
	AccountCount int                  `json:"accountCount"`
	EntryCount   int                  `json:"entryCount"`
	Accounts     []SeedPreviewAccount `json:"accounts"`
}

type SeedPreviewRequest struct {
	Profile string `query:"profile" validate:"required"`
	Lang    string `query:"lang" validate:"required"`
}
