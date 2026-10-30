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

	roles := reg.Roles()
	var viewer, admin *authz.Role
	for i := range roles {
		switch roles[i].ID {
		case domain.RoleViewer:
			viewer = &roles[i]
		case domain.RoleAdmin:
			admin = &roles[i]
		}
	}

	require.NotNil(t, viewer)
	require.NotNil(t, admin)

	// Viewer checks
	assert.Contains(t, viewer.Permissions, authz.Permission{Object: domain.ObjProvider, Action: "read"})
	assert.Len(t, viewer.Permissions, 1, "viewer should have only read on provider")
	for _, p := range viewer.Permissions {
		assert.NotEqual(t, "core:file", p.Object)
	}

	// Admin checks
	assert.Contains(t, admin.Inherits, domain.RoleViewer)
	assert.Contains(t, admin.Permissions, authz.Permission{Object: "agent:*", Action: authz.Wildcard})
	for _, p := range admin.Permissions {
		assert.NotEqual(t, "core:file", p.Object)
	}

	// Validate registry integrity
	require.NoError(t, reg.Validate())
}
