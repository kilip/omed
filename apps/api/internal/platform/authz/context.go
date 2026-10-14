package authz

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

// MEMBERSHIP_CONTEXT_KEY is the context key used to store the resolved workspace Membership.
const MEMBERSHIP_CONTEXT_KEY = "authz.membership"

// Membership is the resolved authorization state of the current user in a workspace.
type Membership struct {
	WorkspaceID int64
	UserID      int64
	IsOwner     bool
	Roles       []string
}

// MembershipResolver resolves the membership of a user in a workspace.
// Implementations must return base.ErrNotFound when the workspace does not exist
// or the user is neither its owner nor a member.
type MembershipResolver interface {
	ResolveMembership(ctx context.Context, userID, workspaceID int64) (Membership, error)
}

// WithMembership stores the membership in the Fiber request locals and sets base workspace ID.
func WithMembership(c fiber.Ctx, m Membership) {
	c.Locals(MEMBERSHIP_CONTEXT_KEY, m)
	if m.WorkspaceID > 0 {
		c.Locals(base.WORKSPACE_CONTEXT_KEY, m.WorkspaceID)
	}
}

// ContextWithMembership returns a child context carrying the membership and workspace ID.
// Useful for service-level checks outside an HTTP request (e.g. tests, jobs).
func ContextWithMembership(ctx context.Context, m Membership) context.Context {
	ctx = context.WithValue(ctx, MEMBERSHIP_CONTEXT_KEY, m) //nolint:staticcheck // string key mirrors base.USER_CONTEXT_KEY
	if m.WorkspaceID > 0 {
		ctx = base.WithWorkspaceID(ctx, m.WorkspaceID)
	}
	return ctx
}

// MembershipFromContext retrieves the Membership resolved by the authz middleware, if present.
func MembershipFromContext(ctx context.Context) (Membership, bool) {
	if ctx == nil {
		return Membership{}, false
	}
	if fctx, ok := ctx.(fiber.Ctx); ok {
		if m, ok := fctx.Locals(MEMBERSHIP_CONTEXT_KEY).(Membership); ok {
			return m, true
		}
	}
	if m, ok := ctx.Value(MEMBERSHIP_CONTEXT_KEY).(Membership); ok {
		return m, true
	}
	return Membership{}, false
}
