package domain

import "github.com/kilip/omed/api/internal/platform/authz"

// Fin module permission objects.
const (
	// ObjAccount is the permission object for Chart of Accounts management.
	ObjAccount = "fin:account"
)

// Built-in fin roles assignable per workspace.
const (
	// RoleViewer can view accounts and reports.
	RoleViewer = "fin:viewer"
	// RoleAccountant can manage accounts, transactions, and journals.
	RoleAccountant = "fin:accountant"
	// RoleAdmin has full permissions across all financial operations.
	RoleAdmin = "fin:admin"
)

func init() {
	authz.AddRole(authz.Role{
		Name:        RoleViewer,
		Description: "View accounts, reports, and financial summaries",
		Permissions: []authz.Permission{
			{Object: ObjAccount, Action: "read"},
		},
	})
	authz.AddRole(authz.Role{
		Name:        RoleAccountant,
		Description: "Manage accounts, journal entries, and transactions",
		Inherits:    []string{RoleViewer},
		Permissions: []authz.Permission{
			{Object: ObjAccount, Action: "create"},
			{Object: ObjAccount, Action: "update"},
		},
	})
	authz.AddRole(authz.Role{
		Name:        RoleAdmin,
		Description: "Full administrative access to financial module",
		Inherits:    []string{RoleAccountant},
		Permissions: []authz.Permission{
			{Object: ObjAccount, Action: "delete"},
			{Object: "fin:*", Action: authz.Wildcard},
		},
	})
}
