package base

import (
	"context"

	"github.com/gofiber/fiber/v3"
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

// UserFromContext retrieves the authenticated UserContext from a context.Context, if present.
func UserFromContext(ctx context.Context) (UserContext, bool) {
	if ctx == nil {
		return UserContext{}, false
	}
	if fctx, ok := ctx.(fiber.Ctx); ok {
		val := fctx.Locals(USER_CONTEXT_KEY)
		if u, ok := val.(UserContext); ok {
			return u, true
		}
		if u, ok := val.(*UserContext); ok && u != nil {
			return *u, true
		}
	}

	if u, ok := ctx.Value(USER_CONTEXT_KEY).(UserContext); ok {
		return u, true
	}
	if u, ok := ctx.Value(USER_CONTEXT_KEY).(*UserContext); ok && u != nil {
		return *u, true
	}
	return UserContext{}, false
}
