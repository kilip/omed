// Package authz provides workspace-scoped, role-based authorization backed by Casbin.
//
// Every module declares its own fixed roles in code, typically from an init() function,
// following the same pattern as base.AddToErrorMap:
//
//	func init() {
//		authz.AddRole(authz.Role{
//			Name:        "fin:accountant",
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
	roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*:[a-z][a-z0-9_]*$`)
	objectPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*:([a-z][a-z0-9_]*|\*)$`)
	actionPattern   = regexp.MustCompile(`^([a-z][a-z0-9_]*|\*)$`)
)

// Permission is a single (object, action) pair, e.g. {"fin:invoice", "read"}.
type Permission struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

// Role is a module-scoped role definition, e.g. "fin:accountant".
type Role struct {
	// Name is the fully qualified role name in the form "<module>:<role>".
	Name string
	// Description is a human readable description of the role.
	Description string
	// Permissions granted directly by this role.
	Permissions []Permission
	// Inherits lists other role names whose permissions are granted by this role.
	Inherits []string
}

// Module returns the module prefix of the role name ("fin" for "fin:accountant").
func (r Role) Module() string {
	mod, _, _ := strings.Cut(r.Name, ":")
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

// Add validates and registers a role. It returns an error on duplicate names or invalid format.
func (r *Registry) Add(role Role) error {
	if err := validateRole(role); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.roles[role.Name]; exists {
		return fmt.Errorf("authz: role %q already registered", role.Name)
	}
	r.roles[role.Name] = role
	return nil
}

// Has reports whether a role with the given name is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.roles[name]
	return ok
}

// Roles returns all registered roles sorted by name.
func (r *Registry) Roles() []Role {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Role, 0, len(r.roles))
	for _, role := range r.roles {
		out = append(out, role)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
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
				return fmt.Errorf("authz: role %q inherits unknown role %q", role.Name, parent)
			}
		}
	}
	return nil
}

func validateRole(role Role) error {
	if !roleNamePattern.MatchString(role.Name) {
		return fmt.Errorf("authz: invalid role name %q, expected <module>:<role>", role.Name)
	}
	mod := role.Module()
	for _, p := range role.Permissions {
		if p.Object != Wildcard {
			if !objectPattern.MatchString(p.Object) {
				return fmt.Errorf("authz: role %q has invalid object %q", role.Name, p.Object)
			}
			if objMod, _, _ := strings.Cut(p.Object, ":"); objMod != mod {
				return fmt.Errorf("authz: role %q cannot grant object %q outside module %q", role.Name, p.Object, mod)
			}
		}
		if !actionPattern.MatchString(p.Action) {
			return fmt.Errorf("authz: role %q has invalid action %q", role.Name, p.Action)
		}
	}
	for _, parent := range role.Inherits {
		if !roleNamePattern.MatchString(parent) {
			return fmt.Errorf("authz: role %q inherits invalid role name %q", role.Name, parent)
		}
		if parent == role.Name {
			return fmt.Errorf("authz: role %q cannot inherit itself", role.Name)
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

// HasRole reports whether a role is registered in the default registry.
func HasRole(name string) bool {
	return defaultRegistry.Has(name)
}

// Roles returns all roles registered in the default registry, sorted by name.
func Roles() []Role {
	return defaultRegistry.Roles()
}

// DefaultRegistry returns the process-wide registry used by AddRole.
func DefaultRegistry() *Registry {
	return defaultRegistry
}
