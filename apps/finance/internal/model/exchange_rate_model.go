package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ExchangeRate struct {
	ID           uuid.UUID       `json:"id"`
	WorkspaceID  uuid.UUID       `json:"workspaceId"`
	FromCurrency string          `json:"fromCurrency"`
	ToCurrency   string          `json:"toCurrency"`
	Rate         decimal.Decimal `json:"rate"`
	RateDate     time.Time       `json:"rateDate"`
	Source       *string         `json:"source,omitempty"`
}

func (r *ExchangeRate) UnmarshalJSON(data []byte) error {
	type Alias ExchangeRate
	var raw struct {
		Alias
		FromCurrencyAlt *string    `json:"from_currency"`
		ToCurrencyAlt   *string    `json:"to_currency"`
		RateDateAlt     *time.Time `json:"rate_date"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = ExchangeRate(raw.Alias)
	if r.FromCurrency == "" && raw.FromCurrencyAlt != nil {
		r.FromCurrency = *raw.FromCurrencyAlt
	}
	if r.ToCurrency == "" && raw.ToCurrencyAlt != nil {
		r.ToCurrency = *raw.ToCurrencyAlt
	}
	if r.RateDate.IsZero() && raw.RateDateAlt != nil {
		r.RateDate = *raw.RateDateAlt
	}
	return nil
}

type CreateExchangeRateRequest struct {
	FromCurrency string          `json:"fromCurrency" validate:"required,len=3,uppercase" example:"USD"`
	ToCurrency   string          `json:"toCurrency" validate:"required,len=3,uppercase" example:"IDR"`
	Rate         decimal.Decimal `json:"rate" validate:"required"`
	RateDate     time.Time       `json:"rateDate" validate:"required"`
	Source       *string         `json:"source,omitempty" validate:"omitempty,max=255"`
}

func (r *CreateExchangeRateRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		FromCurrency    string          `json:"fromCurrency"`
		FromCurrencyAlt string          `json:"from_currency"`
		ToCurrency      string          `json:"toCurrency"`
		ToCurrencyAlt   string          `json:"to_currency"`
		Rate            decimal.Decimal `json:"rate"`
		RateDate        any             `json:"rateDate"`
		RateDateAlt     any             `json:"rate_date"`
		Source          *string         `json:"source"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.FromCurrency = raw.FromCurrency
	if r.FromCurrency == "" {
		r.FromCurrency = raw.FromCurrencyAlt
	}
	r.ToCurrency = raw.ToCurrency
	if r.ToCurrency == "" {
		r.ToCurrency = raw.ToCurrencyAlt
	}
	r.Rate = raw.Rate
	r.Source = raw.Source

	dateVal := raw.RateDate
	if dateVal == nil {
		dateVal = raw.RateDateAlt
	}
	if dateVal != nil {
		t, err := parseFlexibleTime(dateVal)
		if err != nil {
			return err
		}
		r.RateDate = t
	}

	return nil
}

type UpdateExchangeRateRequest struct {
	Rate   *decimal.Decimal `json:"rate,omitempty"`
	Source *string          `json:"source,omitempty" validate:"omitempty,max=255"`
}

type ListExchangeRateRequest struct {
	FromCurrency string `query:"from_currency" validate:"omitempty,len=3,uppercase"`
	ToCurrency   string `query:"to_currency" validate:"omitempty,len=3,uppercase"`
	RateDate     string `query:"rate_date"`
}
