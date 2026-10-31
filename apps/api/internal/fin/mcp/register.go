package mcp

import (
	"context"

	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

// Insights defines the service interface required by fin MCP tools.
type Insights interface {
	PeriodOverview(context.Context, model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error)
	AccountBalance(context.Context, model.AccountBalanceRequest) (model.AccountBalanceResponse, error)
	FindCashflow(context.Context, model.FindCashflowRequest) (model.FindCashflowResponse, error)
}

// RegisterTools registers the finance insight tools with the MCP registrar.
func RegisterTools(r *mcpx.Registrar, in Insights) {
	registerPeriodOverview(r, in)
	registerAccountBalance(r, in)
	registerFindCashflow(r, in)
}
