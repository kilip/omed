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

// WORKSPACE_CONTEXT_KEY is the context key used to store and retrieve the active workspace ID.
const WORKSPACE_CONTEXT_KEY = "workspace_id"

// SKIP_WORKSPACE_FILTER_KEY is the context key used to bypass workspace query filtering.
const SKIP_WORKSPACE_FILTER_KEY = "skip_workspace_filter"

type contextLocalGetter interface {
	Locals(key any, value ...any) any
}

// WithWorkspaceID returns a child context carrying the workspace ID.
func WithWorkspaceID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, WORKSPACE_CONTEXT_KEY, id)
}

// WorkspaceIDFromContext retrieves the workspace ID from a context.Context or fiber.Ctx locals.
func WorkspaceIDFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if fctx, ok := ctx.(contextLocalGetter); ok {
		if id, ok := fctx.Locals(WORKSPACE_CONTEXT_KEY).(int64); ok {
			return id
		}
	}
	if id, ok := ctx.Value(WORKSPACE_CONTEXT_KEY).(int64); ok {
		return id
	}
	return 0
}

// WithSkipWorkspaceFilter returns a child context indicating that workspace query filter should be bypassed.
func WithSkipWorkspaceFilter(ctx context.Context) context.Context {
	return context.WithValue(ctx, SKIP_WORKSPACE_FILTER_KEY, true)
}

// SkipWorkspaceFilterFromContext reports whether the workspace query filter should be bypassed.
func SkipWorkspaceFilterFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if fctx, ok := ctx.(contextLocalGetter); ok {
		if skip, ok := fctx.Locals(SKIP_WORKSPACE_FILTER_KEY).(bool); ok {
			return skip
		}
	}
	if skip, ok := ctx.Value(SKIP_WORKSPACE_FILTER_KEY).(bool); ok {
		return skip
	}
	return false
}

// SKIP_AUDIT_MUTATOR_KEY is the context key used to bypass audit mutator field injection.
const SKIP_AUDIT_MUTATOR_KEY = "skip_audit_mutator"

// WithSkipAuditMutator returns a child context indicating that audit mutator should be bypassed.
func WithSkipAuditMutator(ctx context.Context) context.Context {
	return context.WithValue(ctx, SKIP_AUDIT_MUTATOR_KEY, true)
}

// SkipAuditMutatorFromContext reports whether the audit mutator should be bypassed.
func SkipAuditMutatorFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if fctx, ok := ctx.(contextLocalGetter); ok {
		if skip, ok := fctx.Locals(SKIP_AUDIT_MUTATOR_KEY).(bool); ok {
			return skip
		}
	}
	if skip, ok := ctx.Value(SKIP_AUDIT_MUTATOR_KEY).(bool); ok {
		return skip
	}
	return false
}
