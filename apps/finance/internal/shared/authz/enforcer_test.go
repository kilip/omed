package authz_test

import (
	"testing"

	"github.com/kilip/omed/finance/internal/shared"
	"github.com/kilip/omed/finance/internal/shared/authz"
	"github.com/stretchr/testify/assert"
)

func TestPolicy(t *testing.T) {
	enforcer := authz.GetEnforcer()
	ok, err := enforcer.Enforce("owner", shared.GenerateID().String(), string(authz.ResourceUsers), string(authz.ActionRead))

	assert.Nil(t, err)
	assert.True(t, ok)
}
