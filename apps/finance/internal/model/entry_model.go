package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EntryType string

const (
	EntryTypeNormal         EntryType = "normal"
	EntryTypeOpeningBalance EntryType = "opening_balance"
	EntryTypeAdjustment     EntryType = "adjustment"
	EntryTypeClosing        EntryType = "closing"
	EntryTypeFXAdjustment   EntryType = "fx_adjustment"
)

type Posting struct {
	ID               uuid.UUID       `json:"id"`
	WorkspaceID      uuid.UUID       `json:"workspaceId"`
	EntryID          uuid.UUID       `json:"entryId"`
	AccountID        uuid.UUID       `json:"accountId"`
	AccountCode      string          `json:"accountCode,omitempty"`
	AccountName      string          `json:"accountName,omitempty"`
	Currency         string          `json:"currency"`
	DebitAmount      decimal.Decimal `json:"debitAmount"`
	CreditAmount     decimal.Decimal `json:"creditAmount"`
	BaseCurrency     string          `json:"baseCurrency"`
	BaseDebitAmount  decimal.Decimal `json:"baseDebitAmount"`
	BaseCreditAmount decimal.Decimal `json:"baseCreditAmount"`
	ExchangeRate     decimal.Decimal `json:"exchangeRate"`
	Memo             *string         `json:"memo,omitempty"`
}

func (p *Posting) UnmarshalJSON(data []byte) error {
	type Alias Posting
	var raw struct {
		Alias
		EntryIDAlt          *uuid.UUID       `json:"entry_id"`
		AccountIDAlt        *uuid.UUID       `json:"account_id"`
		DebitAmountAlt      *decimal.Decimal `json:"debit_amount"`
		CreditAmountAlt     *decimal.Decimal `json:"credit_amount"`
		BaseCurrencyAlt     *string          `json:"base_currency"`
		BaseDebitAmountAlt  *decimal.Decimal `json:"base_debit_amount"`
		BaseCreditAmountAlt *decimal.Decimal `json:"base_credit_amount"`
		ExchangeRateAlt     *decimal.Decimal `json:"exchange_rate"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = Posting(raw.Alias)
	if p.EntryID == uuid.Nil && raw.EntryIDAlt != nil {
		p.EntryID = *raw.EntryIDAlt
	}
	if p.AccountID == uuid.Nil && raw.AccountIDAlt != nil {
		p.AccountID = *raw.AccountIDAlt
	}
	if p.DebitAmount.IsZero() && raw.DebitAmountAlt != nil {
		p.DebitAmount = *raw.DebitAmountAlt
	}
	if p.CreditAmount.IsZero() && raw.CreditAmountAlt != nil {
		p.CreditAmount = *raw.CreditAmountAlt
	}
	if p.BaseCurrency == "" && raw.BaseCurrencyAlt != nil {
		p.BaseCurrency = *raw.BaseCurrencyAlt
	}
	if p.BaseDebitAmount.IsZero() && raw.BaseDebitAmountAlt != nil {
		p.BaseDebitAmount = *raw.BaseDebitAmountAlt
	}
	if p.BaseCreditAmount.IsZero() && raw.BaseCreditAmountAlt != nil {
		p.BaseCreditAmount = *raw.BaseCreditAmountAlt
	}
	if p.ExchangeRate.IsZero() && raw.ExchangeRateAlt != nil {
		p.ExchangeRate = *raw.ExchangeRateAlt
	}
	return nil
}

type Entry struct {
	ID             uuid.UUID       `json:"id"`
	WorkspaceID    uuid.UUID       `json:"workspaceId"`
	EntryDate      time.Time       `json:"entryDate"`
	EntryType      EntryType       `json:"entryType" enums:"normal,opening_balance,adjustment,closing,fx_adjustment"`
	Description    string          `json:"description,omitempty"`
	Reference      *string         `json:"reference,omitempty"`
	LedgerPeriodID uuid.UUID       `json:"ledgerPeriodId"`
	Postings       []Posting       `json:"postings,omitempty"`
	CreatedBy      uuid.UUID       `json:"createdBy"`
	CreatedByName  string          `json:"createdByName"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedBy      uuid.UUID       `json:"updatedBy"`
	UpdatedByName  string          `json:"updatedByName"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	type Alias Entry
	var raw struct {
		Alias
		EntryDateAlt      *time.Time `json:"entry_date"`
		EntryTypeAlt      *EntryType `json:"entry_type"`
		LedgerPeriodIDAlt *uuid.UUID `json:"ledger_period_id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*e = Entry(raw.Alias)
	if e.EntryDate.IsZero() && raw.EntryDateAlt != nil {
		e.EntryDate = *raw.EntryDateAlt
	}
	if e.EntryType == "" && raw.EntryTypeAlt != nil {
		e.EntryType = *raw.EntryTypeAlt
	}
	if e.LedgerPeriodID == uuid.Nil && raw.LedgerPeriodIDAlt != nil {
		e.LedgerPeriodID = *raw.LedgerPeriodIDAlt
	}
	return nil
}

type CreatePostingRequest struct {
	AccountID    uuid.UUID       `json:"accountId" validate:"required"`
	Currency     string          `json:"currency" validate:"required,len=3,uppercase" example:"IDR"`
	DebitAmount  decimal.Decimal `json:"debitAmount"`
	CreditAmount decimal.Decimal `json:"creditAmount"`
	Memo         *string         `json:"memo,omitempty" validate:"omitempty,max=1000"`
}

func (r *CreatePostingRequest) UnmarshalJSON(data []byte) error {
	type Alias CreatePostingRequest
	var raw struct {
		Alias
		AccountIDAlt    *uuid.UUID       `json:"account_id"`
		DebitAmountAlt  *decimal.Decimal `json:"debit_amount"`
		CreditAmountAlt *decimal.Decimal `json:"credit_amount"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = CreatePostingRequest(raw.Alias)
	if r.AccountID == uuid.Nil && raw.AccountIDAlt != nil {
		r.AccountID = *raw.AccountIDAlt
	}
	if r.DebitAmount.IsZero() && raw.DebitAmountAlt != nil {
		r.DebitAmount = *raw.DebitAmountAlt
	}
	if r.CreditAmount.IsZero() && raw.CreditAmountAlt != nil {
		r.CreditAmount = *raw.CreditAmountAlt
	}
	return nil
}

type CreateEntryRequest struct {
	EntryDate   time.Time              `json:"entryDate" validate:"required"`
	EntryType   EntryType              `json:"entryType,omitempty" validate:"omitempty,oneof=normal opening_balance adjustment closing fx_adjustment" enums:"normal,opening_balance,adjustment,closing,fx_adjustment"`
	Description string                 `json:"description,omitempty" validate:"omitempty,max=1000"`
	Reference   *string                `json:"reference,omitempty" validate:"omitempty,max=255"`
	Postings    []CreatePostingRequest `json:"postings" validate:"required,min=2,dive"`
}

func (r *CreateEntryRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		EntryDate    any                    `json:"entryDate"`
		EntryDateAlt any                    `json:"entry_date"`
		EntryType    EntryType              `json:"entryType"`
		EntryTypeAlt EntryType              `json:"entry_type"`
		Description  string                 `json:"description"`
		Reference    *string                `json:"reference"`
		Postings     []CreatePostingRequest `json:"postings"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Description = raw.Description
	r.Reference = raw.Reference
	r.Postings = raw.Postings

	r.EntryType = raw.EntryType
	if r.EntryType == "" {
		r.EntryType = raw.EntryTypeAlt
	}

	dateVal := raw.EntryDate
	if dateVal == nil {
		dateVal = raw.EntryDateAlt
	}
	if dateVal != nil {
		t, err := parseFlexibleTime(dateVal)
		if err != nil {
			return err
		}
		r.EntryDate = t
	}

	return nil
}

type ListEntryRequest struct {
	Cursor    string    `query:"cursor"`
	Limit     int       `query:"limit" validate:"omitempty,min=1,max=100"`
	StartDate string    `query:"start_date"`
	EndDate   string    `query:"end_date"`
	EntryType EntryType `query:"entry_type" validate:"omitempty,oneof=normal opening_balance adjustment closing fx_adjustment"`
}
