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

// CmsPostTranslation holds the schema definition for the CmsPostTranslation entity.
type CmsPostTranslation struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CmsPostTranslation table.
func (CmsPostTranslation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("cms"),
	}
}

// Mixin of the CmsPostTranslation entity.
func (CmsPostTranslation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
		PublishMixin{},
	}
}

// Fields of the CmsPostTranslation entity.
func (CmsPostTranslation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("post_id"),
		field.String("locale").NotEmpty(),
		field.String("title").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.String("excerpt").Optional(),
		field.Text("body").Default(""),
	}
}

// Edges of the CmsPostTranslation entity.
func (CmsPostTranslation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("post", CmsPost.Type).
			Ref("translations").
			Field("post_id").
			Unique().
			Required(),
	}
}

// Indexes of the CmsPostTranslation entity.
func (CmsPostTranslation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("post_id", "locale").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "locale", "slug").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "locale", "status", "publish_at"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
