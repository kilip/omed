package base_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/internal/base"
	"github.com/stretchr/testify/assert"
)

func TestUserFromContext(t *testing.T) {
	expected := base.UserContext{
		ID:            42,
		Name:          "Alice",
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	}
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, expected)
	u := base.UserFromContext(ctx)
	assert.Equal(t, expected, u)
}

func TestUserContext_HasRole(t *testing.T) {
	u := base.UserContext{Roles: []base.UserRole{base.UserRoleUser}}
	assert.True(t, u.HasRole(base.UserRoleUser))
	assert.False(t, u.HasRole(base.UserRole("other")))
	assert.False(t, base.UserContext{}.HasRole(base.UserRoleUser))
}

type mockLocalGetter struct {
	context.Context
	locals map[string]any
}

func (m mockLocalGetter) Locals(key any, value ...any) any {
	if s, ok := key.(string); ok {
		return m.locals[s]
	}
	if ck, ok := key.(base.ContextKey); ok {
		return m.locals[string(ck)]
	}
	return nil
}

func TestWorkspaceContext(t *testing.T) {
	// Nil context
	var nilCtx context.Context
	assert.Equal(t, int64(0), base.WorkspaceIDFromContext(nilCtx))
	assert.False(t, base.SkipWorkspaceFilterFromContext(nilCtx))

	// Standard context
	ctx := context.Background()
	assert.Equal(t, int64(0), base.WorkspaceIDFromContext(ctx))
	assert.False(t, base.SkipWorkspaceFilterFromContext(ctx))

	ctx = base.WithWorkspaceID(ctx, 123)
	assert.Equal(t, int64(123), base.WorkspaceIDFromContext(ctx))

	ctx = base.WithSkipWorkspaceFilter(ctx)
	assert.True(t, base.SkipWorkspaceFilterFromContext(ctx))

	// Local getter context (like fiber.Ctx)
	fctx := mockLocalGetter{
		Context: context.Background(),
		locals: map[string]any{
			string(base.WORKSPACE_CONTEXT_KEY):     int64(456),
			string(base.SKIP_WORKSPACE_FILTER_KEY): true,
		},
	}
	assert.Equal(t, int64(456), base.WorkspaceIDFromContext(fctx))
	assert.True(t, base.SkipWorkspaceFilterFromContext(fctx))
}
