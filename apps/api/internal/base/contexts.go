package base

import (
	"context"
)

// USER_CONTEXT_KEY is the context key used to store and retrieve authenticated user details.
const USER_CONTEXT_KEY = "user"

// UserRole represents the authorization role assigned to a user.
type UserRole string

const (
	// UserRoleAdmin represents a super administrator role with elevated system privileges.
	UserRoleAdmin UserRole = "superadmin"
	// UserRoleUser represents a standard authenticated user.
	UserRoleUser UserRole = "user"
)

// UserContext contains the authenticated user's identity, verification status, and assigned roles.
type UserContext struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	EmailVerified bool       `json:"emailVerified"`
	Roles         []UserRole `json:"roles"`
}

// HasRole reports whether the user has the given global role (from the JWT roles claim).
func (u UserContext) HasRole(role UserRole) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// UserFromContext retrieves the authenticated UserContext from a context.Context.
// UserContext always exists, otherwise there's something wrong with fiber middleware
func UserFromContext(ctx context.Context) UserContext {
	return ctx.Value(USER_CONTEXT_KEY).(UserContext)
}
