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
	// ObjFile is the global permission object for workspace file management.
	ObjFile = "core:file"
)

// Built-in core role IDs assignable per workspace.
const (
	// RoleMember can view the workspace and its members.
	RoleMember = "core:member"
	// RoleAdmin can manage the workspace settings and its members.
	RoleAdmin = "core:admin"
)

func init() {
	authz.AddRole(authz.Role{
		ID:          RoleMember,
		Name:        "Member",
		Description: "View the workspace and its members",
		Permissions: []authz.Permission{
			{Object: ObjWorkspace, Action: "read"},
			{Object: ObjMember, Action: "read"},
			{Object: ObjFile, Action: "read"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleAdmin,
		Name:        "Admin",
		Description: "Manage the workspace settings and its members",
		Inherits:    []string{RoleMember},
		Permissions: []authz.Permission{
			{Object: "core:*", Action: authz.Wildcard},
		},
	})
}
