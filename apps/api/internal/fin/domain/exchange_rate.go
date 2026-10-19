// Package domain defines the domain models and business errors for the finance module.
package domain

import (
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// ExchangeRateSource defines the source mechanism for currency exchange rates.
type ExchangeRateSource string

const (
	// ExchangeRateSourceManual indicates the exchange rate was entered manually.
	ExchangeRateSourceManual ExchangeRateSource = "MANUAL"
	// ExchangeRateSourceImport indicates the exchange rate was fetched via automatic/external feed.
	ExchangeRateSourceImport ExchangeRateSource = "IMPORT"
)

// ExchangeRate represents a currency conversion rate effective on a specific date for a workspace.
type ExchangeRate struct {
	ID            int64              `json:"id"`
	WorkspaceID   int64              `json:"workspaceId"`
	FromCurrency  string             `json:"fromCurrency"`
	ToCurrency    string             `json:"toCurrency"`
	Rate          float64            `json:"rate"`
	EffectiveDate time.Time          `json:"effectiveDate"`
	Source        ExchangeRateSource `json:"source"`
	CreatedByID   int64              `json:"createdById"`
	CreatedBy     base.ByUser        `json:"createdBy"`
	UpdatedByID   int64              `json:"updatedById"`
	UpdatedBy     base.ByUser        `json:"updatedBy"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
	DeletedAt     *time.Time         `json:"deletedAt,omitempty"`
}
