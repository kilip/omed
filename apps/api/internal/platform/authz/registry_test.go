package authz_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// finRoles is a fixture module proving that a module can declare its own roles.
var finRoles = []authz.Role{
	{
		ID:          "fin:viewer",
		Name:        "Viewer",
		Permissions: []authz.Permission{{Object: "fin:report", Action: "read"}},
	},
	{
		ID:          "fin:accountant",
		Name:        "Accountant",
		Inherits:    []string{"fin:viewer"},
		Permissions: []authz.Permission{{Object: "fin:invoice", Action: "read"}, {Object: "fin:invoice", Action: "create"}},
	},
	{
		ID:          "fin:admin",
		Name:        "Admin",
		Permissions: []authz.Permission{{Object: "fin:*", Action: "*"}},
	},
}

func newFinRegistry(t *testing.T) *authz.Registry {
	t.Helper()
	reg := authz.NewRegistry()
	for _, r := range finRoles {
		require.NoError(t, reg.Add(r))
	}
	return reg
}

func TestRegistry_Add_Validation(t *testing.T) {
	cases := map[string]authz.Role{
		"missing module":      {ID: "accountant"},
		"uppercase":           {ID: "Fin:Accountant"},
		"display name as id":  {ID: "Accountant"},
		"empty id":            {Name: "Accountant"},
		"foreign object":      {ID: "fin:x", Permissions: []authz.Permission{{Object: "core:member", Action: "read"}}},
		"invalid object":      {ID: "fin:x", Permissions: []authz.Permission{{Object: "invoice", Action: "read"}}},
		"invalid action":      {ID: "fin:x", Permissions: []authz.Permission{{Object: "fin:invoice", Action: "Read!"}}},
		"self inherit":        {ID: "fin:x", Inherits: []string{"fin:x"}},
		"invalid parent id":   {ID: "fin:x", Inherits: []string{"bad"}},
		"parent is a display": {ID: "fin:x", Inherits: []string{"Viewer"}},
	}
	for name, role := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, authz.NewRegistry().Add(role))
		})
	}
}

func TestRegistry_Add_AllowsWildcards(t *testing.T) {
	reg := authz.NewRegistry()
	assert.NoError(t, reg.Add(authz.Role{ID: "fin:root", Permissions: []authz.Permission{{Object: "*", Action: "*"}}}))
	assert.NoError(t, reg.Add(authz.Role{ID: "fin:mod", Permissions: []authz.Permission{{Object: "fin:*", Action: "read"}}}))
}

func TestRegistry_Add_Duplicate(t *testing.T) {
	reg := authz.NewRegistry()
	require.NoError(t, reg.Add(authz.Role{ID: "fin:viewer"}))
	assert.Error(t, reg.Add(authz.Role{ID: "fin:viewer"}))
}

func TestRegistry_Add_DisplayNameIsFreeText(t *testing.T) {
	reg := authz.NewRegistry()
	// Name is presentation only: it may contain spaces and capitals, and two roles may share it.
	require.NoError(t, reg.Add(authz.Role{ID: "fin:viewer", Name: "Read Only"}))
	require.NoError(t, reg.Add(authz.Role{ID: "core:member", Name: "Read Only"}))
}

func TestRegistry_HasAndRolesSorted(t *testing.T) {
	reg := newFinRegistry(t)
	assert.True(t, reg.Has("fin:accountant"))
	assert.False(t, reg.Has("fin:unknown"))
	assert.False(t, reg.Has("Accountant"), "lookup is by ID, not display name")

	roles := reg.Roles()
	require.Len(t, roles, 3)
	assert.Equal(t, "fin:accountant", roles[0].ID)
	assert.Equal(t, "Accountant", roles[0].Name)
	assert.Equal(t, "fin:admin", roles[1].ID)
	assert.Equal(t, "fin:viewer", roles[2].ID)
	assert.Equal(t, "fin", roles[0].Module())
}

func TestRegistry_Validate_UnknownParent(t *testing.T) {
	reg := authz.NewRegistry()
	require.NoError(t, reg.Add(authz.Role{ID: "fin:x", Inherits: []string{"fin:missing"}}))
	assert.Error(t, reg.Validate())

	_, err := authz.NewEnforcer(reg)
	assert.Error(t, err)
}

func TestAddRole_PanicsOnInvalid(t *testing.T) {
	assert.Panics(t, func() { authz.AddRole(authz.Role{ID: "invalid"}) })
}

func TestDefaultRegistry(t *testing.T) {
	authz.AddRole(authz.Role{ID: "testmod:reader", Name: "Reader", Permissions: []authz.Permission{{Object: "testmod:doc", Action: "read"}}})
	assert.True(t, authz.HasRole("testmod:reader"))
	assert.Same(t, authz.DefaultRegistry(), authz.DefaultRegistry())

	found := false
	for _, r := range authz.Roles() {
		if r.ID == "testmod:reader" {
			found = true
			assert.Equal(t, "Reader", r.Name)
		}
	}
	assert.True(t, found)
}
