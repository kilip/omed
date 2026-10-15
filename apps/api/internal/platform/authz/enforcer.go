package authz

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
)

//go:embed model.conf
var modelConf string

// Enforcer is an in-memory Casbin enforcer built from role definitions.
// It evaluates role IDs (not users) against objects and actions.
type Enforcer struct {
	e *casbin.Enforcer
}

// NewEnforcer builds an enforcer from the roles in the given registry.
func NewEnforcer(reg *Registry) (*Enforcer, error) {
	if err := reg.Validate(); err != nil {
		return nil, err
	}

	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("authz: load model: %w", err)
	}
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("authz: create enforcer: %w", err)
	}

	var policies, groupings [][]string
	for _, role := range reg.Roles() {
		for _, p := range role.Permissions {
			policies = append(policies, []string{role.ID, p.Object, p.Action})
		}
		for _, parent := range role.Inherits {
			groupings = append(groupings, []string{role.ID, parent})
		}
	}
	if len(policies) > 0 {
		if _, err := e.AddPolicies(policies); err != nil {
			return nil, fmt.Errorf("authz: add policies: %w", err)
		}
	}
	if len(groupings) > 0 {
		if _, err := e.AddGroupingPolicies(groupings); err != nil {
			return nil, fmt.Errorf("authz: add role inheritance: %w", err)
		}
	}
	return &Enforcer{e: e}, nil
}

// Allowed reports whether any of the given roles grants the action on the object.
func (e *Enforcer) Allowed(roles []string, obj, act string) bool {
	for _, role := range roles {
		ok, err := e.e.Enforce(role, obj, act)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// Permissions returns the deduplicated effective permissions granted by the given roles,
// including those inherited from parent roles, sorted by object then action.
func (e *Enforcer) Permissions(roles []string) []Permission {
	seen := map[Permission]struct{}{}
	for _, role := range roles {
		rules, err := e.e.GetImplicitPermissionsForUser(role)
		if err != nil {
			continue
		}
		for _, rule := range rules {
			if len(rule) < 3 {
				continue
			}
			seen[Permission{Object: rule[1], Action: rule[2]}] = struct{}{}
		}
	}
	out := make([]Permission, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Object != out[j].Object {
			return out[i].Object < out[j].Object
		}
		return out[i].Action < out[j].Action
	})
	return out
}
