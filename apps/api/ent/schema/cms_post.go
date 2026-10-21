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
		entsql.Schema("omed"),
	}
}

// Mixin of the CmsPost entity.
func (CmsPost) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
		PublishMixin{},
	}
}

// Fields of the CmsPost entity.
func (CmsPost) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("title").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.String("excerpt").Optional(),
		field.Text("body").Default(""),
		field.Int64("cover_media_id").Optional().Nillable(),
	}
}

// Edges of the CmsPost entity.
func (CmsPost) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cover_media", CmsMedia.Type).
			Field("cover_media_id").
			Unique(),
	}
}

// Indexes of the CmsPost entity.
func (CmsPost) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "slug").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "status", "publish_at"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
