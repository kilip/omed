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

// Workspace holds the schema definition for the Workspace entity.
type Workspace struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Workspace table.
func (Workspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("core"),
	}
}

// Mixin of the Workspace entity.
func (Workspace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		AuditMixin{},
	}
}

// Fields of the Workspace entity.
func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.String("description").Optional(),
		field.Int64("owner_id"),
		field.Bool("personal").Default(false).Immutable(),
		field.String("base_currency").MinLen(3).MaxLen(3).Default("IDR"),
	}
}

// Edges of the Workspace entity.
func (Workspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owner", User.Type).
			Field("owner_id").
			Unique().
			Required(),
		edge.To("members", WorkspaceMember.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the Workspace entity.
func (Workspace) Indexes() []ent.Index {
	return []ent.Index{
		// Every user owns at most one personal workspace.
		index.Fields("owner_id").
			Unique().
			Annotations(entsql.IndexWhere("personal")).
			StorageKey("workspace_personal_owner"),
	}
}
