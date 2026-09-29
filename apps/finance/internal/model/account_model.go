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
type Account struct {
	ID            uuid.UUID     `json:"id"`
	WorkspaceID   uuid.UUID     `json:"workspaceId"`
	Code          string        `json:"code"`
	Name          string        `json:"name"`
	Description   *string       `json:"description,omitempty"`
	Type          AccountType   `json:"type"`
	Currency      string        `json:"currency"`
	Status        AccountStatus `json:"status"`
	ParentID      *uuid.UUID    `json:"parentId,omitempty"`
	CreatedBy     uuid.UUID     `json:"createdBy"`
	CreatedByName string        `json:"createdByName"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedBy     uuid.UUID     `json:"updatedBy"`
	UpdatedByName string        `json:"updatedByName"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type CreateAccountRequest struct {
	Code        string      `json:"code"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Type        AccountType `json:"type"`
	Currency    string      `json:"currency"`
	ParentID    *uuid.UUID  `json:"parentId,omitempty"`
}

// UpdateAccountRequest: code, type & currency sengaja tidak bisa diubah.
type UpdateAccountRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Status      *AccountStatus `json:"status,omitempty"`
	ParentID    *uuid.UUID     `json:"parentId,omitempty"`
}

type ListAccountRequest struct {
	Type   AccountType   `query:"type"`
	Status AccountStatus `query:"status"`
}
