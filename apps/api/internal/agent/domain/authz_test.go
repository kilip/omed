package domain_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRolesRegistered(t *testing.T) {
	reg := authz.DefaultRegistry()

	require.True(t, reg.Has(domain.RoleViewer), "expected RoleViewer to be registered")
	require.True(t, reg.Has(domain.RoleAdmin), "expected RoleAdmin to be registered")
	require.True(t, reg.Has(domain.RoleFinancialUser), "expected RoleFinancialUser to be registered")

	roles := reg.Roles()
	var viewer, admin, finUser *authz.Role
	for i := range roles {
		switch roles[i].ID {
		case domain.RoleViewer:
			viewer = &roles[i]
		case domain.RoleAdmin:
			admin = &roles[i]
		case domain.RoleFinancialUser:
			finUser = &roles[i]
		}
	}

	require.NotNil(t, viewer)
	require.NotNil(t, admin)
	require.NotNil(t, finUser)

	// Viewer checks: read provider and read setting
	assert.Contains(t, viewer.Permissions, authz.Permission{Object: domain.ObjProvider, Action: "read"})
	assert.Contains(t, viewer.Permissions, authz.Permission{Object: domain.ObjSetting, Action: "read"})
	assert.Len(t, viewer.Permissions, 2, "viewer should have only read on provider and setting")
	for _, p := range viewer.Permissions {
		assert.NotEqual(t, "core:file", p.Object)
	}

	// Financial user checks: use financial and nothing else; does not inherit and is not inherited
	assert.Equal(t, []authz.Permission{{Object: domain.ObjFinancial, Action: "use"}}, finUser.Permissions)
	assert.Empty(t, finUser.Inherits, "financial user should not inherit any roles")

	// Admin checks: inherits viewer, wildcard on agent:*
	assert.Contains(t, admin.Inherits, domain.RoleViewer)
	assert.NotContains(t, admin.Inherits, domain.RoleFinancialUser, "admin does not directly inherit financial user")
	assert.Contains(t, admin.Permissions, authz.Permission{Object: "agent:*", Action: authz.Wildcard})
	for _, p := range admin.Permissions {
		assert.NotEqual(t, "core:file", p.Object)
	}

	// Validate registry integrity
	require.NoError(t, reg.Validate())

	// Enforcer checks
	enf, err := authz.NewEnforcer(reg)
	require.NoError(t, err)

	// agent:viewer can read agent:setting
	assert.True(t, enf.Allowed([]string{domain.RoleViewer}, domain.ObjSetting, "read"))
	// agent:viewer cannot use agent:financial
	assert.False(t, enf.Allowed([]string{domain.RoleViewer}, domain.ObjFinancial, "use"))

	// agent:financial-user can use agent:financial and nothing else
	assert.True(t, enf.Allowed([]string{domain.RoleFinancialUser}, domain.ObjFinancial, "use"))
	assert.False(t, enf.Allowed([]string{domain.RoleFinancialUser}, domain.ObjSetting, "read"))
	assert.False(t, enf.Allowed([]string{domain.RoleFinancialUser}, domain.ObjSetting, "update"))
	assert.False(t, enf.Allowed([]string{domain.RoleFinancialUser}, domain.ObjProvider, "read"))

	// agent:admin can update agent:setting and use agent:financial
	assert.True(t, enf.Allowed([]string{domain.RoleAdmin}, domain.ObjSetting, "update"))
	assert.True(t, enf.Allowed([]string{domain.RoleAdmin}, domain.ObjFinancial, "use"))
}
