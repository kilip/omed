package domain_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmsRoles_RegistrationAndInheritance(t *testing.T) {
	reg := authz.DefaultRegistry()
	require.NoError(t, reg.Validate())

	require.True(t, reg.Has(domain.RoleViewer))
	require.True(t, reg.Has(domain.RoleEditor))
	require.True(t, reg.Has(domain.RoleAdmin))

	roles := reg.Roles()
	var viewer, editor, admin *authz.Role
	for i := range roles {
		switch roles[i].ID {
		case domain.RoleViewer:
			viewer = &roles[i]
		case domain.RoleEditor:
			editor = &roles[i]
		case domain.RoleAdmin:
			admin = &roles[i]
		}
	}

	require.NotNil(t, viewer)
	require.NotNil(t, editor)
	require.NotNil(t, admin)

	assert.Empty(t, viewer.Inherits)
	assert.Equal(t, []string{domain.RoleViewer}, editor.Inherits)
	assert.Equal(t, []string{domain.RoleEditor}, admin.Inherits)

	// Check viewer permissions
	expectedViewerPerms := []authz.Permission{
		{Object: domain.ObjPost, Action: "read"},
		{Object: domain.ObjPage, Action: "read"},
		{Object: domain.ObjMedia, Action: "read"},
	}
	assert.ElementsMatch(t, expectedViewerPerms, viewer.Permissions)

	// Check admin wildcard permission
	expectedAdminPerms := []authz.Permission{
		{Object: "cms:*", Action: "*"},
	}
	assert.ElementsMatch(t, expectedAdminPerms, admin.Permissions)
}
