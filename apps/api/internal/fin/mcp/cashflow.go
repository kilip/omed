package mcp

import (
	"context"

	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

const cashflowDescription = "Searches income or expense cashflow documents in a workspace by type ('income' or 'expense'), date range (YYYY-MM-DD format), status ('draft' or 'posted'), text query, or account references. Returns up to 50 items (default 20) with pagination cursor."

func registerFindCashflow(r *mcpx.Registrar, in Insights) {
	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_find_cashflow",
		Description: cashflowDescription,
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:transaction", Action: "read"},
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, req model.FindCashflowRequest) (model.FindCashflowResponse, error) {
		return in.FindCashflow(ctx, req)
	})
}
