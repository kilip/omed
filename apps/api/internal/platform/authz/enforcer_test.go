package authz_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnforcer_Allowed(t *testing.T) {
	enf, err := authz.NewEnforcer(newFinRegistry(t))
	require.NoError(t, err)

	cases := []struct {
		name  string
		roles []string
		obj   string
		act   string
		want  bool
	}{
		{"direct permission", []string{"fin:accountant"}, "fin:invoice", "read", true},
		{"missing action", []string{"fin:accountant"}, "fin:invoice", "delete", false},
		{"inherited permission", []string{"fin:accountant"}, "fin:report", "read", true},
		{"viewer cannot read invoice", []string{"fin:viewer"}, "fin:invoice", "read", false},
		{"wildcard object and action", []string{"fin:admin"}, "fin:invoice", "delete", true},
		{"wildcard does not cross module", []string{"fin:admin"}, "core:member", "read", false},
		{"any of many roles", []string{"fin:viewer", "fin:admin"}, "fin:ledger", "update", true},
		{"no roles", nil, "fin:invoice", "read", false},
		{"unknown role", []string{"fin:ghost"}, "fin:invoice", "read", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, enf.Allowed(tc.roles, tc.obj, tc.act))
		})
	}
}

func TestEnforcer_GlobalWildcard(t *testing.T) {
	reg := authz.NewRegistry()
	require.NoError(t, reg.Add(authz.Role{Name: "fin:root", Permissions: []authz.Permission{{Object: "*", Action: "*"}}}))
	enf, err := authz.NewEnforcer(reg)
	require.NoError(t, err)
	assert.True(t, enf.Allowed([]string{"fin:root"}, "anything:here", "whatever"))
}

func TestEnforcer_Permissions(t *testing.T) {
	enf, err := authz.NewEnforcer(newFinRegistry(t))
	require.NoError(t, err)

	perms := enf.Permissions([]string{"fin:accountant", "fin:viewer"})
	assert.Equal(t, []authz.Permission{
		{Object: "fin:invoice", Action: "create"},
		{Object: "fin:invoice", Action: "read"},
		{Object: "fin:report", Action: "read"},
	}, perms)

	assert.Empty(t, enf.Permissions(nil))
}

func TestEnforcer_EmptyRegistry(t *testing.T) {
	enf, err := authz.NewEnforcer(authz.NewRegistry())
	require.NoError(t, err)
	assert.False(t, enf.Allowed([]string{"fin:accountant"}, "fin:invoice", "read"))
}
