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
	AccountTypeIncome    AccountType = "income"
	AccountTypeExpense   AccountType = "expense"
)

// cash|bank|ewallet|credit_card|payable|receivable
type AccountDetailType string

const (
	AccountDetailTypeCash       AccountDetailType = "cash"
	AccountDetailTypeBank       AccountDetailType = "bank"
	AccountDetailTypeEwallet    AccountDetailType = "ewallet"
	AccountDetailTypeCreditCard AccountDetailType = "credit-card"
	AccountDetailTypePayable    AccountDetailType = "payable"
	AccountDetailTypeReceivable AccountDetailType = "receivable"
)

type AccountNormalBalance string

const (
	AccountNormalBalanceDebit  AccountNormalBalance = "debit"
	AccountNormalBalanceCredit AccountNormalBalance = "credit"
)

type AccountResponse struct {
	ID            uuid.UUID            `json:"id"`
	ParentID      *uuid.UUID           `json:"parentId,omitempty"`
	Code          string               `json:"code"`
	Name          string               `json:"name"`
	Type          AccountType          `json:"type"`
	DetailType    AccountDetailType    `json:"detailType"`
	NormalBalance AccountNormalBalance `json:"normalBalance"`
	Currency      string               `json:"currency"`
	Active        bool                 `json:"active"`
	Placeholder   bool                 `json:"placeholder"`
	System        bool                 `json:"system"`
	CreatedBy     uuid.UUID            `json:"createdBy,omitzero"`
	CreatedAt     time.Time            `json:"createdAt,omitzero"`
	UpdatedBy     uuid.UUID            `json:"updatedBy,omitzero"`
	UpdatedAt     time.Time            `json:"updatedAt,omitzero"`
}

type AccountRequest struct {
	ParentID      *uuid.UUID           `json:"parentId,omitempty"`
	Code          string               `json:"code"`
	Name          string               `json:"name"`
	Type          AccountType          `json:"type"`
	DetailType    AccountDetailType    `json:"detailType"`
	NormalBalance AccountNormalBalance `json:"normalBalance"`
	Currency      string               `json:"currency"`
	Active        bool                 `json:"active"`
	Placeholder   bool                 `json:"placeholder"`
	System        bool                 `json:"system"`
}

type AccountFilter struct {
	ParentID *uuid.UUID
	Type     *AccountType
}
