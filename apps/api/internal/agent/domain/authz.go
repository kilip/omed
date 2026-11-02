package domain

import "github.com/kilip/omed/api/internal/platform/authz"

// Agent module permission objects.
const (
	ObjProvider  = "agent:provider"
	ObjSetting   = "agent:setting"
	ObjFinancial = "agent:financial"
)

// Built-in agent role IDs assignable per workspace.
const (
	RoleViewer        = "agent:viewer"
	RoleFinancialUser = "agent:financial-user"
	RoleAdmin         = "agent:admin"
)

func init() {
	authz.AddRole(authz.Role{
		ID:          RoleViewer,
		Name:        "Agent Viewer",
		Description: "View agent providers and configurations",
		Permissions: []authz.Permission{
			{Object: ObjProvider, Action: "read"},
			{Object: ObjSetting, Action: "read"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleFinancialUser,
		Name:        "Agent Financial User",
		Description: "Use financial agent analysis features",
		Permissions: []authz.Permission{
			{Object: ObjFinancial, Action: "use"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleAdmin,
		Name:        "Agent Admin",
		Description: "Full administrative access to agent module",
		Inherits:    []string{RoleViewer},
		Permissions: []authz.Permission{
			{Object: "agent:*", Action: authz.Wildcard},
		},
	})
}
