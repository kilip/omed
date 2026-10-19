// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// CreateExchangeRateRequest defines the payload required to create a new exchange rate.
type CreateExchangeRateRequest struct {
	FromCurrency  string  `json:"fromCurrency" validate:"required,len=3" example:"USD"`
	ToCurrency    string  `json:"toCurrency" validate:"required,len=3" example:"IDR"`
	Rate          float64 `json:"rate" validate:"required,gt=0" example:"16250.0"`
	EffectiveDate Date    `json:"effectiveDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Source        *string `json:"source,omitempty" validate:"omitempty,oneof=MANUAL IMPORT" example:"MANUAL"`
}

// UpdateExchangeRateRequest defines the payload required to update an existing exchange rate.
type UpdateExchangeRateRequest struct {
	Rate          float64 `json:"rate" validate:"required,gt=0" example:"16300.0"`
	EffectiveDate Date    `json:"effectiveDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	Source        *string `json:"source,omitempty" validate:"omitempty,oneof=MANUAL IMPORT" example:"MANUAL"`
}

// SearchExchangeRateRequest defines filter criteria and pagination options for exchange rates.
type SearchExchangeRateRequest struct {
	FromCurrency *string `json:"fromCurrency,omitempty" example:"USD"`
	ToCurrency   *string `json:"toCurrency,omitempty" example:"IDR"`
	StartDate    *Date   `json:"startDate,omitempty" swaggertype:"string" example:"2026-01-01"`
	EndDate      *Date   `json:"endDate,omitempty" swaggertype:"string" example:"2026-12-31"`
	Cursor       string  `json:"cursor,omitempty"`
	Limit        int     `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// ExchangeRateResponse represents exchange rate metadata returned by API endpoints.
type ExchangeRateResponse struct {
	ID            int64       `json:"id"`
	WorkspaceID   int64       `json:"workspaceId"`
	FromCurrency  string      `json:"fromCurrency"`
	ToCurrency    string      `json:"toCurrency"`
	Rate          float64     `json:"rate"`
	EffectiveDate Date        `json:"effectiveDate" swaggertype:"string" example:"2026-01-01"`
	Source        string      `json:"source"`
	CreatedBy     base.ByUser `json:"createdBy"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedBy     base.ByUser `json:"updatedBy"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

// ExchangeRateLookupResponse represents the response when querying the best exchange rate for a currency pair.
type ExchangeRateLookupResponse struct {
	FromCurrency string    `json:"fromCurrency" example:"USD"`
	ToCurrency   string    `json:"toCurrency" example:"IDR"`
	Rate         float64   `json:"rate" example:"16250.0"`
	Date         time.Time `json:"date"`
}
