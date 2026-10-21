package domain

import "github.com/kilip/omed/api/internal/platform/authz"

// CMS module permission objects.
const (
	// ObjPost is the permission object for Post management.
	ObjPost = "cms:post"
	// ObjPage is the permission object for Page management.
	ObjPage = "cms:page"
	// ObjMedia is the permission object for Media library management.
	ObjMedia = "cms:media"
)

// Built-in CMS role IDs assignable per workspace.
const (
	// RoleViewer can view posts, pages, and media.
	RoleViewer = "cms:viewer"
	// RoleEditor can create, update, delete, and publish posts, pages, and media.
	RoleEditor = "cms:editor"
	// RoleAdmin has full administrative access to CMS operations.
	RoleAdmin = "cms:admin"
)

func init() {
	authz.AddRole(authz.Role{
		ID:          RoleViewer,
		Name:        "Viewer",
		Description: "View posts, pages, and media",
		Permissions: []authz.Permission{
			{Object: ObjPost, Action: "read"},
			{Object: ObjPage, Action: "read"},
			{Object: ObjMedia, Action: "read"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleEditor,
		Name:        "Editor",
		Description: "Create, update, delete, and publish posts, pages, and media",
		Inherits:    []string{RoleViewer},
		Permissions: []authz.Permission{
			{Object: ObjPost, Action: "create"},
			{Object: ObjPost, Action: "update"},
			{Object: ObjPost, Action: "delete"},
			{Object: ObjPage, Action: "create"},
			{Object: ObjPage, Action: "update"},
			{Object: ObjPage, Action: "delete"},
			{Object: ObjMedia, Action: "create"},
			{Object: ObjMedia, Action: "update"},
			{Object: ObjMedia, Action: "delete"},
		},
	})
	authz.AddRole(authz.Role{
		ID:          RoleAdmin,
		Name:        "Admin",
		Description: "Full administrative access to CMS module",
		Inherits:    []string{RoleEditor},
		Permissions: []authz.Permission{
			{Object: "cms:*", Action: authz.Wildcard},
		},
	})
}
