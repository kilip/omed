package authz

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

// WorkspaceParam is the route parameter holding the workspace ID.
const WorkspaceParam = "wid"

// Authorizer enforces workspace-scoped permissions for HTTP routes and services.
type Authorizer struct {
	enforcer *Enforcer
	resolver MembershipResolver
	registry *Registry
}

// New creates an Authorizer from the default registry (roles registered via AddRole).
func New(resolver MembershipResolver) (*Authorizer, error) {
	return NewWithRegistry(defaultRegistry, resolver)
}

// NewWithRegistry creates an Authorizer from a specific registry.
func NewWithRegistry(reg *Registry, resolver MembershipResolver) (*Authorizer, error) {
	enf, err := NewEnforcer(reg)
	if err != nil {
		return nil, err
	}
	return &Authorizer{enforcer: enf, resolver: resolver, registry: reg}, nil
}

// HasRole reports whether a role exists in the registry backing this Authorizer.
func (a *Authorizer) HasRole(name string) bool {
	return a.registry.Has(name)
}

// Roles returns the roles in the registry backing this Authorizer, sorted by ID.
func (a *Authorizer) Roles() []Role {
	return a.registry.Roles()
}

// resolve loads the current user's membership for the :wid route param and stores it in context.
func (a *Authorizer) resolve(c fiber.Ctx) (Membership, error) {
	raw := c.Params(WorkspaceParam)
	wid, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || wid <= 0 {
		return Membership{}, base.ErrInvalidID
	}

	if m, ok := MembershipFromContext(c); ok && m.WorkspaceID == wid {
		return m, nil
	}

	user := base.UserFromContext(c)
	if user.ID == 0 {
		return Membership{}, base.ErrUnauthorized
	}

	m, err := a.resolver.ResolveMembership(c, user.ID, wid)
	if err != nil {
		return Membership{}, err
	}
	WithMembership(c, m)
	return m, nil
}

// Member requires the current user to be the owner or a member of the :wid workspace.
// Non-members receive 404 so that workspace existence is not leaked.
func (a *Authorizer) Member() fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, err := a.resolve(c); err != nil {
			return err
		}
		return c.Next()
	}
}

// Require requires the current user to hold a role in the :wid workspace that grants
// the action on the object. Workspace owners are always allowed.
func (a *Authorizer) Require(obj, act string) fiber.Handler {
	return func(c fiber.Ctx) error {
		m, err := a.resolve(c)
		if err != nil {
			return err
		}
		if !a.allowed(m, obj, act) {
			return base.ErrForbidden
		}
		return c.Next()
	}
}

// RequireOwner allows only the owner of the :wid workspace.
func (a *Authorizer) RequireOwner() fiber.Handler {
	return func(c fiber.Ctx) error {
		m, err := a.resolve(c)
		if err != nil {
			return err
		}
		if !m.IsOwner {
			return base.ErrForbidden
		}
		return c.Next()
	}
}

// Can reports whether the membership stored in ctx grants the action on the object.
// It returns false when no membership has been resolved.
func (a *Authorizer) Can(ctx context.Context, obj, act string) bool {
	m, ok := MembershipFromContext(ctx)
	if !ok {
		return false
	}
	return a.allowed(m, obj, act)
}

// Permissions returns the effective permissions of the membership stored in ctx.
// Owners receive a single wildcard permission.
func (a *Authorizer) Permissions(ctx context.Context) []Permission {
	m, ok := MembershipFromContext(ctx)
	if !ok {
		return []Permission{}
	}
	if m.IsOwner {
		return []Permission{{Object: Wildcard, Action: Wildcard}}
	}
	return a.enforcer.Permissions(m.Roles)
}

func (a *Authorizer) allowed(m Membership, obj, act string) bool {
	if m.IsOwner {
		return true
	}
	return a.enforcer.Allowed(m.Roles, obj, act)
}
