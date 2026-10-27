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

// CmsMedia holds the schema definition for the CmsMedia entity.
type CmsMedia struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CmsMedia table.
func (CmsMedia) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("cms"),
	}
}

// Mixin of the CmsMedia entity.
func (CmsMedia) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the CmsMedia entity.
func (CmsMedia) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("file_id"),
		field.String("title").Optional(),
		field.String("alt_text").Optional(),
		field.String("caption").Optional(),
		field.Bool("is_public").Default(true),
	}
}

// Edges of the CmsMedia entity.
func (CmsMedia) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("file", File.Type).
			Field("file_id").
			Unique().
			Required(),
	}
}

// Indexes of the CmsMedia entity.
func (CmsMedia) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "file_id"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
