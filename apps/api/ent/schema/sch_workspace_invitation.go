// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkspaceInvitation holds the schema definition for the WorkspaceInvitation entity.
type WorkspaceInvitation struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the WorkspaceInvitation table.
func (WorkspaceInvitation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the WorkspaceInvitation entity.
func (WorkspaceInvitation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the WorkspaceInvitation entity.
func (WorkspaceInvitation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("token").NotEmpty().Unique(),
		field.Strings("roles").Default([]string{}),
		field.Int("max_uses").Default(0),
		field.Int("uses_count").Default(0),
		field.Time("expires_at").Optional().Nillable(),
		field.Bool("is_revoked").Default(false),
	}
}

// Indexes of the WorkspaceInvitation entity.
func (WorkspaceInvitation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token").Unique(),
		index.Fields("workspace_id", "is_revoked"),
	}
}
