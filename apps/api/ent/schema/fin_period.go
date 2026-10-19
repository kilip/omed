// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Period holds the schema definition for the Period entity (Accounting Period).
type Period struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Period table.
func (Period) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the Period entity.
func (Period) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the Period entity.
func (Period) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.Time("start_date"),
		field.Time("end_date"),
		field.Int("fiscal_year"),
		field.Enum("status").Values("OPEN", "CLOSED").Default("OPEN"),
		field.String("description").Optional(),
	}
}

// Edges of the Period entity.
func (Period) Edges() []ent.Edge {
	return nil
}

// Indexes of the Period entity.
func (Period) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "fiscal_year"),
		index.Fields("workspace_id", "status"),
		index.Fields("workspace_id", "deleted_at"),
		index.Fields("workspace_id", "start_date", "end_date"),
	}
}
