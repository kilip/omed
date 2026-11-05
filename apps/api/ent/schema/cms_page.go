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

// CmsPage holds the schema definition for the CmsPage entity.
type CmsPage struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CmsPage table.
func (CmsPage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("cms"),
	}
}

// Mixin of the CmsPage entity.
func (CmsPage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the CmsPage entity.
func (CmsPage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
	}
}

// Edges of the CmsPage entity.
func (CmsPage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("translations", CmsPageTranslation.Type),
	}
}

// Indexes of the CmsPage entity.
func (CmsPage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "deleted_at"),
	}
}
