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
		Roles:         []base.UserRole{base.UserRoleAdmin},
	}
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, expected)
	u := base.UserFromContext(ctx)
	assert.Equal(t, expected, u)
}

func TestUserContext_HasRole(t *testing.T) {
	u := base.UserContext{Roles: []base.UserRole{base.UserRoleUser, base.UserRoleAdmin}}
	assert.True(t, u.HasRole(base.UserRoleAdmin))
	assert.True(t, u.HasRole(base.UserRoleUser))
	assert.False(t, base.UserContext{}.HasRole(base.UserRoleAdmin))
	assert.False(t, base.UserContext{Roles: []base.UserRole{base.UserRoleUser}}.HasRole(base.UserRoleAdmin))
}
