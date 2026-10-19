// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkspaceMember holds the schema definition for a user's membership in a workspace.
// Roles are module-scoped role names (e.g. "fin:accountant") declared in code via authz.AddRole.
type WorkspaceMember struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the WorkspaceMember table.
func (WorkspaceMember) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the WorkspaceMember entity.
func (WorkspaceMember) Mixin() []ent.Mixin {
	return []ent.Mixin{
		AuditMixin{},
	}
}

// Fields of the WorkspaceMember entity.
func (WorkspaceMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("workspace_id").Immutable(),
		field.Int64("user_id").Immutable(),
		field.Strings("roles").Default([]string{}),
	}
}

// Edges of the WorkspaceMember entity.
func (WorkspaceMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("members").
			Field("workspace_id").
			Unique().
			Required().
			Immutable(),
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the WorkspaceMember entity.
func (WorkspaceMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "user_id").Unique(),
		index.Fields("user_id"),
	}
}
