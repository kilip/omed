package domain

import "github.com/kilip/omed/api/internal/platform/authz"

// Core module objects used in permissions.
const (
	// ObjWorkspace is the permission object for workspace settings.
	ObjWorkspace = "core:workspace"
	// ObjMember is the permission object for workspace membership management.
	ObjMember = "core:member"
	// ObjInvitation is the permission object for workspace invitation management.
	ObjInvitation = "core:invitation"
)

// Built-in core roles assignable per workspace.
const (
	// RoleMember can view the workspace and its members.
	RoleMember = "core:member"
	// RoleAdmin can manage the workspace settings and its members.
	RoleAdmin = "core:admin"
)

func init() {
	authz.AddRole(authz.Role{
		Name:        RoleMember,
		Description: "View the workspace and its members",
		Permissions: []authz.Permission{
			{Object: ObjWorkspace, Action: "read"},
			{Object: ObjMember, Action: "read"},
		},
	})
	authz.AddRole(authz.Role{
		Name:        RoleAdmin,
		Description: "Manage the workspace settings and its members",
		Inherits:    []string{RoleMember},
		Permissions: []authz.Permission{
			{Object: "core:*", Action: authz.Wildcard},
		},
	})
}
