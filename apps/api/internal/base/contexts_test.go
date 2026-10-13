package base_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/internal/base"
	"github.com/stretchr/testify/assert"
)

func TestUserFromContext(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		u, ok := base.UserFromContext(nil)
		assert.False(t, ok)
		assert.Equal(t, base.UserContext{}, u)
	})

	t.Run("context without user", func(t *testing.T) {
		u, ok := base.UserFromContext(context.Background())
		assert.False(t, ok)
		assert.Equal(t, base.UserContext{}, u)
	})

	t.Run("context with value", func(t *testing.T) {
		expected := base.UserContext{
			ID:            42,
			Name:          "Alice",
			EmailVerified: true,
			Roles:         []base.UserRole{base.UserRoleAdmin},
		}
		ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, expected)
		u, ok := base.UserFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, expected, u)
	})

	t.Run("context with pointer value", func(t *testing.T) {
		expected := base.UserContext{
			ID:            42,
			Name:          "Alice",
			EmailVerified: true,
			Roles:         []base.UserRole{base.UserRoleUser},
		}
		ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, &expected)
		u, ok := base.UserFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, expected, u)
	})
}

func TestUserContext_HasRole(t *testing.T) {
	u := base.UserContext{Roles: []base.UserRole{base.UserRoleUser, base.UserRoleAdmin}}
	assert.True(t, u.HasRole(base.UserRoleAdmin))
	assert.True(t, u.HasRole(base.UserRoleUser))
	assert.False(t, base.UserContext{}.HasRole(base.UserRoleAdmin))
	assert.False(t, base.UserContext{Roles: []base.UserRole{base.UserRoleUser}}.HasRole(base.UserRoleAdmin))
}
