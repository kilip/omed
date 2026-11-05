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

// CmsPageTranslation holds the schema definition for the CmsPageTranslation entity.
type CmsPageTranslation struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CmsPageTranslation table.
func (CmsPageTranslation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("cms"),
	}
}

// Mixin of the CmsPageTranslation entity.
func (CmsPageTranslation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
		PublishMixin{},
	}
}

// Fields of the CmsPageTranslation entity.
func (CmsPageTranslation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("page_id"),
		field.String("locale").NotEmpty(),
		field.String("title").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.Text("body").Default(""),
	}
}

// Edges of the CmsPageTranslation entity.
func (CmsPageTranslation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("page", CmsPage.Type).
			Ref("translations").
			Field("page_id").
			Unique().
			Required(),
	}
}

// Indexes of the CmsPageTranslation entity.
func (CmsPageTranslation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("page_id", "locale").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "locale", "slug").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "locale", "status", "publish_at"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
