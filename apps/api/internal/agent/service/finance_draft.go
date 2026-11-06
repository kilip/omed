package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
)

// CurrencyOption represents a currency available in the workspace.
type CurrencyOption struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// PeriodOption represents an accounting period in the workspace.
type PeriodOption struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	FiscalYear int       `json:"fiscalYear"`
	Status     string    `json:"status"`
}

// FinanceDraftContext defines the operations required by agent services to read reference
// finance data and persist confirmed draft proposals without posting them.
type FinanceDraftContext interface {
	ListAccounts(ctx context.Context) ([]AccountOption, error)
	ListCurrencies(ctx context.Context) ([]CurrencyOption, error)
	ListPeriods(ctx context.Context) ([]PeriodOption, error)
	SaveDraft(ctx context.Context, proposal model.DraftProposal) (int64, error)
}
