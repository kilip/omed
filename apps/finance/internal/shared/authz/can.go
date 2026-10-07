package authz

import "github.com/kilip/omed/finance/internal/core"

// Can checks if the authenticated user has permission to perform act on obj within their active workspace.
func Can(user core.AuthenticatedUser, obj Resource, act Action) bool {
	enforcer := GetEnforcer()
	wsID := user.WorkspaceID.String()

	for _, role := range user.WorkspaceRoles {
		ok, _ := enforcer.Enforce(string(role), wsID, string(obj), string(act))
		if ok {
			return true
		}
	}
	return false
}
