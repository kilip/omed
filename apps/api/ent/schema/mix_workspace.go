// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// WorkspaceMixin provides workspace tenancy fields and relations (workspace_id, workspace).
type WorkspaceMixin struct {
	mixin.Schema
}

// Fields of the WorkspaceMixin.
func (WorkspaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id").Immutable(),
	}
}

// Edges of the WorkspaceMixin.
func (WorkspaceMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("workspace", Workspace.Type).
			Field("workspace_id").
			Unique().
			Required().
			Immutable(),
	}
}
