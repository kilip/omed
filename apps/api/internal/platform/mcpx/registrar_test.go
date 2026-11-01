package mcpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

type stubResolver struct {
	resolve func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error)
}

func (r *stubResolver) ResolveMembership(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
	if r.resolve != nil {
		return r.resolve(ctx, userID, workspaceID)
	}
	return authz.Membership{}, nil
}

type testInput struct {
	WorkspaceID int64 `json:"workspace_id" jsonschema:"Target workspace ID"`
}

func (in testInput) GetWorkspaceID() int64 {
	return in.WorkspaceID
}

type testOutput struct {
	Message string `json:"message"`
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

func Test_AddTool_PanicsWithoutPermissions(t *testing.T) {
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	require.Panics(t, func() {
		mcpx.AddTool(r, mcpx.ToolSpec{
			Name:        "fin_balance",
			Description: "Account balance",
			ReadOnly:    true,
			Permissions: nil,
		}, func(ctx context.Context, in testInput) (testOutput, error) {
			return testOutput{}, nil
		})
	})
}

func Test_AddTool_PanicsOnUnknownPermissionObject(t *testing.T) {
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	require.Panics(t, func() {
		mcpx.AddTool(r, mcpx.ToolSpec{
			Name:        "fin_balance",
			Description: "Account balance",
			ReadOnly:    true,
			Permissions: []mcpx.Permission{
				{Object: "fin:acount", Action: "read"},
			},
		}, func(ctx context.Context, in testInput) (testOutput, error) {
			return testOutput{}, nil
		})
	})
}

func Test_AddTool_PanicsWhenNameLacksModulePrefix(t *testing.T) {
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	require.Panics(t, func() {
		mcpx.AddTool(r, mcpx.ToolSpec{
			Name:        "balance",
			Description: "Account balance",
			ReadOnly:    true,
			Permissions: []mcpx.Permission{
				{Object: "fin:account", Action: "read"},
			},
		}, func(ctx context.Context, in testInput) (testOutput, error) {
			return testOutput{}, nil
		})
	})
}

func Test_ListTools_ReadOnlyHintAndRequiredWorkspaceID(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	var logBuf bytes.Buffer
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		Logger:     slog.New(slog.NewTextHandler(&logBuf, nil)),
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_balance",
		Description: "Account balance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (testOutput, error) {
		return testOutput{Message: "ok"}, nil
	})

	ct, st := mcp.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	listRes, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, listRes.Tools, 1)

	tool := listRes.Tools[0]
	require.Equal(t, "fin_balance", tool.Name)
	require.NotNil(t, tool.Annotations)
	require.True(t, tool.Annotations.ReadOnlyHint)

	schemaBytes, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	var schemaMap map[string]any
	err = json.Unmarshal(schemaBytes, &schemaMap)
	require.NoError(t, err)

	required, ok := schemaMap["required"].([]any)
	require.True(t, ok)
	require.Contains(t, required, "workspace_id")
}
