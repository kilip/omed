package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	finmcp "github.com/kilip/omed/api/internal/fin/mcp"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func extractErrorPayload(t *testing.T, res *sdk.CallToolResult) errorPayload {
	t.Helper()
	require.NotNil(t, res)
	require.NotEmpty(t, res.Content)
	textContent, ok := res.Content[0].(*sdk.TextContent)
	require.True(t, ok)
	var p errorPayload
	err := json.Unmarshal([]byte(textContent.Text), &p)
	require.NoError(t, err, "expected JSON error payload, got: %s", textContent.Text)
	return p
}

type stubResolver struct {
	resolve func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error)
}

func (s *stubResolver) ResolveMembership(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
	if s.resolve != nil {
		return s.resolve(ctx, userID, workspaceID)
	}
	return authz.Membership{}, nil
}

func setupTestAuthz(resolver authz.MembershipResolver) *authz.Authorizer {
	reg := authz.NewRegistry()
	_ = reg.Add(authz.Role{
		ID:   "fin:viewer",
		Name: "Finance Viewer",
		Permissions: []authz.Permission{
			{Object: "fin:account", Action: "read"},
		},
	})
	_ = reg.Add(authz.Role{
		ID:   "fin:tx-only",
		Name: "Finance Transactions Only",
		Permissions: []authz.Permission{
			{Object: "fin:transaction", Action: "read"},
		},
	})
	_ = reg.Add(authz.Role{
		ID:   "fin:auditor",
		Name: "Finance Auditor",
		Permissions: []authz.Permission{
			{Object: "fin:account", Action: "read"},
			{Object: "fin:transaction", Action: "read"},
		},
	})
	auth, err := authz.NewWithRegistry(reg, resolver)
	if err != nil {
		panic(err)
	}
	return auth
}

type fakeInsights struct {
	periodOverviewFn func(context.Context, model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error)
	accountBalanceFn func(context.Context, model.AccountBalanceRequest) (model.AccountBalanceResponse, error)
	findCashflowFn   func(context.Context, model.FindCashflowRequest) (model.FindCashflowResponse, error)
}

func (f *fakeInsights) PeriodOverview(ctx context.Context, req model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error) {
	if f.periodOverviewFn != nil {
		return f.periodOverviewFn(ctx, req)
	}
	return model.PeriodOverviewResponse{}, nil
}

func (f *fakeInsights) AccountBalance(ctx context.Context, req model.AccountBalanceRequest) (model.AccountBalanceResponse, error) {
	if f.accountBalanceFn != nil {
		return f.accountBalanceFn(ctx, req)
	}
	return model.AccountBalanceResponse{}, nil
}

func (f *fakeInsights) FindCashflow(ctx context.Context, req model.FindCashflowRequest) (model.FindCashflowResponse, error) {
	if f.findCashflowFn != nil {
		return f.findCashflowFn(ctx, req)
	}
	return model.FindCashflowResponse{}, nil
}

func setupClient(t *testing.T, ctx context.Context, r *mcpx.Registrar) *sdk.ClientSession {
	t.Helper()
	ct, st := sdk.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cs.Close()
	})
	return cs
}

func Test_RegisterTools_ThreeReadOnlyTools(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	finmcp.RegisterTools(r, &fakeInsights{})

	cs := setupClient(t, ctx, r)

	listRes, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, listRes.Tools, 3)

	toolsByName := make(map[string]*sdk.Tool)
	for _, tool := range listRes.Tools {
		toolsByName[tool.Name] = tool
	}

	expectedNames := []string{"fin_period_overview", "fin_account_balance", "fin_find_cashflow"}
	for _, name := range expectedNames {
		tool, ok := toolsByName[name]
		require.True(t, ok, "missing tool %s", name)
		require.NotNil(t, tool.Annotations)
		require.True(t, tool.Annotations.ReadOnlyHint, "tool %s must be ReadOnlyHint=true", name)

		schemaBytes, err := json.Marshal(tool.InputSchema)
		require.NoError(t, err)
		var schemaMap map[string]any
		err = json.Unmarshal(schemaBytes, &schemaMap)
		require.NoError(t, err)

		required, ok := schemaMap["required"].([]any)
		require.True(t, ok, "tool %s schema must have required list", name)
		require.Contains(t, required, "workspace_id", "tool %s required list must contain workspace_id", name)
	}
}

func Test_FindCashflow_RoleWithoutAccountRead_Forbidden(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{
		resolve: func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
			return authz.Membership{
				WorkspaceID: workspaceID,
				UserID:      userID,
				Roles:       []string{"fin:tx-only"},
			}, nil
		},
	}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		UserID: func(req *sdk.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	finmcp.RegisterTools(r, &fakeInsights{})
	cs := setupClient(t, ctx, r)

	callRes, err := cs.CallTool(ctx, &sdk.CallToolParams{
		Name: "fin_find_cashflow",
		Arguments: map[string]any{
			"workspace_id": 123,
			"type":         "income",
		},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "FORBIDDEN", p.Code)
}

func Test_Tool_NonMember_NotFound(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{
		resolve: func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
			return authz.Membership{}, base.ErrNotFound
		},
	}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		UserID: func(req *sdk.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	finmcp.RegisterTools(r, &fakeInsights{})
	cs := setupClient(t, ctx, r)

	callRes, err := cs.CallTool(ctx, &sdk.CallToolParams{
		Name: "fin_period_overview",
		Arguments: map[string]any{
			"workspace_id": 123,
		},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "NOT_FOUND", p.Code)
}

func Test_Tools_MoneyIsStringWithCurrency(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{
		resolve: func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
			return authz.Membership{
				WorkspaceID: workspaceID,
				UserID:      userID,
				Roles:       []string{"fin:auditor"},
			}, nil
		},
	}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		UserID: func(req *sdk.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	fake := &fakeInsights{
		periodOverviewFn: func(ctx context.Context, req model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error) {
			return model.PeriodOverviewResponse{
				BaseCurrency: "USD",
				Revenue:      decimal.RequireFromString("100.50"),
				Expense:      decimal.RequireFromString("50.25"),
				Net:          decimal.RequireFromString("50.25"),
				Monthly: []model.MonthTotals{
					{
						Month:   "2026-01",
						Revenue: decimal.RequireFromString("100.50"),
						Expense: decimal.RequireFromString("50.25"),
						Net:     decimal.RequireFromString("50.25"),
					},
				},
			}, nil
		},
		accountBalanceFn: func(ctx context.Context, req model.AccountBalanceRequest) (model.AccountBalanceResponse, error) {
			return model.AccountBalanceResponse{
				BaseCurrency: "USD",
				Totals: model.BalanceTotals{
					Debit:  decimal.RequireFromString("100.50"),
					Credit: decimal.RequireFromString("100.50"),
				},
				Accounts: []model.AccountBalanceItem{
					{
						Code:   "1100",
						Name:   "Cash",
						Debit:  decimal.RequireFromString("100.50"),
						Credit: decimal.Zero,
						Net:    decimal.RequireFromString("100.50"),
					},
				},
			}, nil
		},
		findCashflowFn: func(ctx context.Context, req model.FindCashflowRequest) (model.FindCashflowResponse, error) {
			return model.FindCashflowResponse{
				Items: []model.CashflowDoc{
					{
						ID:         1,
						Number:     "IN-001",
						Currency:   "USD",
						Amount:     decimal.RequireFromString("100.50"),
						BaseAmount: decimal.RequireFromString("100.50"),
						Lines: []model.CashflowLine{
							{
								Amount: decimal.RequireFromString("100.50"),
							},
						},
					},
				},
			}, nil
		},
	}

	finmcp.RegisterTools(r, fake)
	cs := setupClient(t, ctx, r)

	// 1. fin_period_overview
	callRes, err := cs.CallTool(ctx, &sdk.CallToolParams{
		Name:      "fin_period_overview",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)
	textContent := callRes.Content[0].(*sdk.TextContent)
	var mOverview map[string]any
	err = json.Unmarshal([]byte(textContent.Text), &mOverview)
	require.NoError(t, err)
	require.Equal(t, "USD", mOverview["base_currency"])
	require.IsType(t, "", mOverview["revenue"])
	require.IsType(t, "", mOverview["expense"])
	require.IsType(t, "", mOverview["net"])
	monthly := mOverview["monthly"].([]any)
	month0 := monthly[0].(map[string]any)
	require.IsType(t, "", month0["revenue"])

	// 2. fin_account_balance
	callRes, err = cs.CallTool(ctx, &sdk.CallToolParams{
		Name:      "fin_account_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)
	textContent = callRes.Content[0].(*sdk.TextContent)
	var mBalance map[string]any
	err = json.Unmarshal([]byte(textContent.Text), &mBalance)
	require.NoError(t, err)
	require.Equal(t, "USD", mBalance["base_currency"])
	totals := mBalance["totals"].(map[string]any)
	require.IsType(t, "", totals["debit"])
	require.IsType(t, "", totals["credit"])
	accs := mBalance["accounts"].([]any)
	acc0 := accs[0].(map[string]any)
	require.IsType(t, "", acc0["debit"])
	require.IsType(t, "", acc0["credit"])
	require.IsType(t, "", acc0["net"])

	// 3. fin_find_cashflow
	callRes, err = cs.CallTool(ctx, &sdk.CallToolParams{
		Name: "fin_find_cashflow",
		Arguments: map[string]any{
			"workspace_id": 123,
			"type":         "income",
		},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)
	textContent = callRes.Content[0].(*sdk.TextContent)
	var mCashflow map[string]any
	err = json.Unmarshal([]byte(textContent.Text), &mCashflow)
	require.NoError(t, err)
	items := mCashflow["items"].([]any)
	it0 := items[0].(map[string]any)
	require.Equal(t, "USD", it0["currency"])
	require.IsType(t, "", it0["amount"])
	require.IsType(t, "", it0["base_amount"])
	lines := it0["lines"].([]any)
	l0 := lines[0].(map[string]any)
	require.IsType(t, "", l0["amount"])
}

func Test_ArgumentError_ReachesClientWithExpectedFormat(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{
		resolve: func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
			return authz.Membership{
				WorkspaceID: workspaceID,
				UserID:      userID,
				Roles:       []string{"fin:viewer"},
			}, nil
		},
	}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		UserID: func(req *sdk.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	fake := &fakeInsights{
		periodOverviewFn: func(ctx context.Context, req model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error) {
			return model.PeriodOverviewResponse{}, domain.ArgumentError{
				Field:  "from_month",
				Expect: "from_month must use the format YYYY-MM",
			}
		},
	}

	finmcp.RegisterTools(r, fake)
	cs := setupClient(t, ctx, r)

	callRes, err := cs.CallTool(ctx, &sdk.CallToolParams{
		Name:      "fin_period_overview",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	textContent := callRes.Content[0].(*sdk.TextContent)
	require.Contains(t, textContent.Text, "from_month must use the format YYYY-MM")
}
