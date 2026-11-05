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

// CmsPost holds the schema definition for the CmsPost entity.
type CmsPost struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CmsPost table.
func (CmsPost) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("cms"),
	}
}

// Mixin of the CmsPost entity.
func (CmsPost) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the CmsPost entity.
func (CmsPost) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("cover_media_id").Optional().Nillable(),
	}
}

// Edges of the CmsPost entity.
func (CmsPost) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cover_media", CmsMedia.Type).
			Field("cover_media_id").
			Unique(),
		edge.To("translations", CmsPostTranslation.Type),
	}
}

// Indexes of the CmsPost entity.
func (CmsPost) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "deleted_at"),
	}
}
