// Package authz provides workspace-scoped, role-based authorization backed by Casbin.
//
// Every module declares its own fixed roles in code, typically from an init() function,
// following the same pattern as base.AddToErrorMap:
//
//	func init() {
//		authz.AddRole(authz.Role{
//			ID:          "fin:accountant",
//			Name:        "Accountant",
//			Permissions: []authz.Permission{{Object: "fin:invoice", Action: "read"}},
//		})
//	}
//
// Role assignments per workspace are not stored in Casbin. They are resolved per request
// through a MembershipResolver, and the in-memory enforcer only decides whether any of
// the member's roles grants the requested (object, action).
package authz

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Wildcard matches any object or action in a permission.
const Wildcard = "*"

var (
	roleIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*:[a-z][a-z0-9_]*$`)
	objectPattern = regexp.MustCompile(`^[a-z][a-z0-9]*:([a-z][a-z0-9_]*|\*)$`)
	actionPattern = regexp.MustCompile(`^([a-z][a-z0-9_]*|\*)$`)
)

// Permission is a single (object, action) pair, e.g. {"fin:invoice", "read"}.
type Permission struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

// Role is a module-scoped role definition, e.g. "fin:accountant".
type Role struct {
	// ID is the stable, fully qualified identifier in the form "<module>:<role>",
	// e.g. "fin:accountant". It is what memberships and invitations store, what Inherits
	// refers to, and what the enforcer evaluates, so it must never change once released.
	ID string
	// Name is the human readable display name shown in the UI, e.g. "Accountant".
	// It is required. Unlike ID it is free text and may be reworded at any time.
	Name string
	// Description is a human readable description of the role.
	Description string
	// Permissions granted directly by this role.
	Permissions []Permission
	// Inherits lists the IDs of other roles whose permissions are granted by this role.
	Inherits []string
}

// Module returns the module prefix of the role ID ("fin" for "fin:accountant").
func (r Role) Module() string {
	mod, _, _ := strings.Cut(r.ID, ":")
	return mod
}

// Registry holds role definitions declared by modules.
type Registry struct {
	mu    sync.RWMutex
	roles map[string]Role
}

// NewRegistry creates an empty role registry.
func NewRegistry() *Registry {
	return &Registry{roles: map[string]Role{}}
}

// Add validates and registers a role. It returns an error on duplicate IDs or invalid format.
func (r *Registry) Add(role Role) error {
	if err := validateRole(role); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.roles[role.ID]; exists {
		return fmt.Errorf("authz: role %q already registered", role.ID)
	}
	r.roles[role.ID] = role
	return nil
}

// Has reports whether a role with the given ID is registered.
func (r *Registry) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.roles[id]
	return ok
}

// Roles returns all registered roles sorted by ID.
func (r *Registry) Roles() []Role {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Role, 0, len(r.roles))
	for _, role := range r.roles {
		out = append(out, role)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Validate checks cross-role constraints that can only be verified once every module
// has registered its roles, such as references in Inherits.
func (r *Registry) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, role := range r.roles {
		for _, parent := range role.Inherits {
			if _, ok := r.roles[parent]; !ok {
				return fmt.Errorf("authz: role %q inherits unknown role %q", role.ID, parent)
			}
		}
	}
	return nil
}

func validateRole(role Role) error {
	if !roleIDPattern.MatchString(role.ID) {
		return fmt.Errorf("authz: invalid role id %q, expected <module>:<role>", role.ID)
	}
	if strings.TrimSpace(role.Name) == "" {
		return fmt.Errorf("authz: role %q requires a display name", role.ID)
	}
	mod := role.Module()
	for _, p := range role.Permissions {
		if p.Object != Wildcard {
			if !objectPattern.MatchString(p.Object) {
				return fmt.Errorf("authz: role %q has invalid object %q", role.ID, p.Object)
			}
			if objMod, _, _ := strings.Cut(p.Object, ":"); objMod != mod && p.Object != "core:file" {
				return fmt.Errorf("authz: role %q cannot grant object %q outside module %q", role.ID, p.Object, mod)
			}
		}
		if !actionPattern.MatchString(p.Action) {
			return fmt.Errorf("authz: role %q has invalid action %q", role.ID, p.Action)
		}
	}
	for _, parent := range role.Inherits {
		if !roleIDPattern.MatchString(parent) {
			return fmt.Errorf("authz: role %q inherits invalid role id %q", role.ID, parent)
		}
		if parent == role.ID {
			return fmt.Errorf("authz: role %q cannot inherit itself", role.ID)
		}
	}
	return nil
}

var defaultRegistry = NewRegistry()

// AddRole registers a role in the default registry. It panics on invalid or duplicate
// definitions, because these are programming errors that must fail fast at startup.
func AddRole(role Role) {
	if err := defaultRegistry.Add(role); err != nil {
		panic(err)
	}
}

// HasRole reports whether a role ID is registered in the default registry.
func HasRole(id string) bool {
	return defaultRegistry.Has(id)
}

// Roles returns all roles registered in the default registry, sorted by ID.
func Roles() []Role {
	return defaultRegistry.Roles()
}

// DefaultRegistry returns the process-wide registry used by AddRole.
func DefaultRegistry() *Registry {
	return defaultRegistry
}
