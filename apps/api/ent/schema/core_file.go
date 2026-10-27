// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// File holds the schema definition for the File entity.
type File struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the File table.
func (File) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("core"),
	}
}

// Mixin of the File entity.
func (File) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the File entity.
func (File) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.String("bucket").NotEmpty().Immutable(),
		field.String("key").NotEmpty().Immutable(),
		field.String("mime_type").NotEmpty(),
		field.Int64("size").NonNegative(),
		field.String("entity_type").Optional().Nillable(),
		field.Int64("entity_id").Optional().Nillable(),
	}
}

// Indexes of the File entity.
func (File) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "key").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "entity_type", "entity_id"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
