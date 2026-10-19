package domain

import "github.com/kilip/omed/api/internal/platform/authz"

// Fin module permission objects.
const (
	// ObjAccount is the permission object for Chart of Accounts management.
	ObjAccount = "fin:account"
	// ObjPeriod is the permission object for Accounting Period management.
	ObjPeriod = "fin:period"
	// ObjTransaction is the permission object for Transaction management.
	ObjTransaction = "fin:transaction"
	// ObjEntry is the permission object for Journal Entry management.
	ObjEntry = "fin:entry"
)

// Built-in fin role IDs assignable per workspace.
const (
	// RoleViewer can view accounts, periods, and reports.
	RoleViewer = "fin:viewer"
	// RoleAccountant can manage accounts, periods, transactions, and journals.
	RoleAccountant = "fin:accountant"
	// RoleAdmin has full permissions across all financial operations.
	RoleAdmin = "fin:admin"
)

func init() {
	authz.AddRole(authz.Role{
		ID:          RoleViewer,
		Name:        "Viewer",
		Description: "View accounts, reports, and financial summaries",
		Permissions: []authz.Permission{
			{Object: ObjAccount, Action: "read"},
			{Object: ObjPeriod, Action: "read"},
			{Object: ObjTransaction, Action: "read"},
			{Object: ObjEntry, Action: "read"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleAccountant,
		Name:        "Accountant",
		Description: "Manage accounts, journal entries, and transactions",
		Inherits:    []string{RoleViewer},
		Permissions: []authz.Permission{
			{Object: ObjAccount, Action: "create"},
			{Object: ObjAccount, Action: "update"},
			{Object: ObjPeriod, Action: "create"},
			{Object: ObjPeriod, Action: "update"},
			{Object: ObjTransaction, Action: "create"},
			{Object: ObjTransaction, Action: "update"},
			{Object: ObjTransaction, Action: "delete"},
			{Object: ObjEntry, Action: "create"},
			{Object: ObjEntry, Action: "update"},
			{Object: ObjEntry, Action: "delete"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleAdmin,
		Name:        "Admin",
		Description: "Full administrative access to financial module",
		Inherits:    []string{RoleAccountant},
		Permissions: []authz.Permission{
			{Object: ObjAccount, Action: "delete"},
			{Object: ObjPeriod, Action: "delete"},
			{Object: "fin:*", Action: authz.Wildcard},
		},
	})
}
