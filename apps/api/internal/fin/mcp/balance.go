package mcp

import (
	"context"

	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

const balanceDescription = "Provides debit, credit, and net balances for chart-of-accounts accounts in a workspace as of a specific date (YYYY-MM-DD format, defaulting to today), with optional account code prefix or name filtering. At most 50 accounts are returned."

func registerAccountBalance(r *mcpx.Registrar, in Insights) {
	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_account_balance",
		Description: balanceDescription,
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, req model.AccountBalanceRequest) (model.AccountBalanceResponse, error) {
		return in.AccountBalance(ctx, req)
	})
}
