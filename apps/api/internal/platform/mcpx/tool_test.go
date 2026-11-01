package mcpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type shrinkableResult struct {
	Items     []string `json:"items"`
	Truncated bool     `json:"truncated"`
	Hint      string   `json:"hint,omitempty"`
}

func (s *shrinkableResult) Shrink() bool {
	if len(s.Items) <= 1 {
		return false
	}
	s.Items = s.Items[:len(s.Items)-1]
	s.Truncated = true
	s.Hint = "narrow your filters"
	return true
}

type detailedError struct {
	err    error
	detail string
}

func (e detailedError) Error() string {
	return e.err.Error()
}

func (e detailedError) Unwrap() error {
	return e.err
}

func (e detailedError) ErrorDetail() string {
	return e.detail
}

func extractErrorPayload(t *testing.T, res *mcp.CallToolResult) errorPayload {
	t.Helper()
	require.NotNil(t, res)
	require.NotEmpty(t, res.Content)
	textContent, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	var p errorPayload
	err := json.Unmarshal([]byte(textContent.Text), &p)
	require.NoError(t, err, "expected JSON error payload, got: %s", textContent.Text)
	return p
}

func Test_Call_NonMember_ReturnsNotFoundToolError(t *testing.T) {
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
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
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

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "NOT_FOUND", p.Code)
}

func Test_Call_MissingOnePermission_ReturnsForbidden(t *testing.T) {
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
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_cashflow",
		Description: "Cashflow search",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
			{Object: "fin:transaction", Action: "read"},
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

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_cashflow",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "FORBIDDEN", p.Code)
}

func Test_Call_WorkspaceIDMissing_ReturnsValidationMessage(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
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

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 0},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "VALIDATION_FAILED", p.Code)
	require.Equal(t, "workspace_id is required and must be a positive integer", p.Message)
}

func Test_Call_NoUser_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 0, false
		},
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

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "UNAUTHORIZED", p.Code)
}

func Test_Call_Success_ContextCarriesMembership(t *testing.T) {
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
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 77, true
		},
	})

	var seenWorkspaceID int64
	var seenMembership authz.Membership
	var membershipOK bool

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_balance",
		Description: "Account balance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (testOutput, error) {
		seenWorkspaceID = base.WorkspaceIDFromContext(ctx)
		seenMembership, membershipOK = authz.MembershipFromContext(ctx)
		return testOutput{Message: "success"}, nil
	})

	ct, st := mcp.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 999},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)
	require.Equal(t, int64(999), seenWorkspaceID)
	require.True(t, membershipOK)
	require.Equal(t, int64(999), seenMembership.WorkspaceID)
	require.Equal(t, int64(77), seenMembership.UserID)
}

func Test_Call_UnknownError_IsGenericAndLogged(t *testing.T) {
	ctx := context.Background()
	var logBuf bytes.Buffer
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
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
		Logger: slog.New(slog.NewTextHandler(&logBuf, nil)),
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_balance",
		Description: "Account balance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (testOutput, error) {
		return testOutput{}, errors.New("pq: secret")
	})

	ct, st := mcp.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "UNKNOWN", p.Code)
	require.NotContains(t, p.Message, "secret")
	require.Contains(t, p.Message, "internal error (ref: ")

	re := regexp.MustCompile(`ref: ([0-9a-fA-F-]+)`)
	matches := re.FindStringSubmatch(p.Message)
	require.Len(t, matches, 2)
	refID := matches[1]

	logOutput := logBuf.String()
	require.Contains(t, logOutput, "secret")
	require.Contains(t, logOutput, refID)
}

func Test_Call_ErrorDetailer_AppendedToMessage(t *testing.T) {
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
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_balance",
		Description: "Account balance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (testOutput, error) {
		return testOutput{}, detailedError{
			err:    base.ErrNotFound,
			detail: "candidates: Cash, Bank",
		}
	})

	ct, st := mcp.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "NOT_FOUND", p.Code)
	require.Contains(t, p.Message, "candidates: Cash, Bank")
}

func Test_Call_OversizeResult_IsShrunkAndFlagged(t *testing.T) {
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
		Authorizer:     auth,
		Resolver:       res,
		MaxResultBytes: 512,
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_balance",
		Description: "Account balance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (*shrinkableResult, error) {
		// Create 10 items of ~100 bytes each, total > 1000 bytes
		items := make([]string, 10)
		for i := range items {
			items[i] = strings.Repeat("item-long-string-", 5)
		}
		return &shrinkableResult{Items: items}, nil
	})

	ct, st := mcp.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)

	var out shrinkableResult
	textContent := callRes.Content[0].(*mcp.TextContent)
	err = json.Unmarshal([]byte(textContent.Text), &out)
	require.NoError(t, err)
	require.True(t, out.Truncated)
	require.NotEmpty(t, out.Hint)
	require.Less(t, len(out.Items), 10)
	require.LessOrEqual(t, len(textContent.Text), 512)
}

func Test_Call_OversizeNotShrinkable_ReturnsResultTooLarge(t *testing.T) {
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
		Authorizer:     auth,
		Resolver:       res,
		MaxResultBytes: 512,
		UserID: func(req *mcp.CallToolRequest) (int64, bool) {
			return 10, true
		},
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_balance",
		Description: "Account balance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (testOutput, error) {
		return testOutput{Message: strings.Repeat("huge-data-", 100)}, nil
	})

	ct, st := mcp.NewInMemoryTransports()
	_, err := r.Server().Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	callRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fin_balance",
		Arguments: map[string]any{"workspace_id": 123},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	p := extractErrorPayload(t, callRes)
	require.Equal(t, "RESULT_TOO_LARGE", p.Code)
}
