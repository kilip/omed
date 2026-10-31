package mcp

import (
	"context"

	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

const overviewDescription = "Provides high-level financial overview (revenue, expenses, net income) for a workspace across a range of calendar months (YYYY-MM format, up to 24 months, defaulting to the current month), with optional comparison against the preceding period of equal length."

func registerPeriodOverview(r *mcpx.Registrar, in Insights) {
	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_period_overview",
		Description: overviewDescription,
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, req model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error) {
		return in.PeriodOverview(ctx, req)
	})
}
