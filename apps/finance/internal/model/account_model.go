package model

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeIncome    AccountType = "income"
	AccountTypeExpense   AccountType = "expense"
)

type Account struct {
	ID       uuid.UUID   `json:"id"`
	ParentID *uuid.UUID  `json:"parentId,omitempty"`
	Code     string      `json:"code"`
	Name     string      `json:"name"`
	Type     AccountType `json:"type"`
	CreatedBy uuid.UUID `json:"createdBy,omitzero"`
	CreatedAt time.Time `json:"createdAt,omitzero"`
	UpdatedBy uuid.UUID `json:"updatedBy,omitzero"`
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
}

type AccountFilter struct {
	ParentID *uuid.UUID
	Type     *AccountType
}

type AccountRepository interface {
	Create(ctx context.Context, r *Account) (*Account, error)
	Update(ctx context.Context, r *Account) (*Account, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter AccountFilter) ([]*Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
}
