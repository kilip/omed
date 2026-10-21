// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
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
		entsql.Schema("omed"),
	}
}

// Mixin of the CmsPage entity.
func (CmsPage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
		PublishMixin{},
	}
}

// Fields of the CmsPage entity.
func (CmsPage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("title").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.Text("body").Default(""),
	}
}

// Edges of the CmsPage entity.
func (CmsPage) Edges() []ent.Edge {
	return nil
}

// Indexes of the CmsPage entity.
func (CmsPage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "slug").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "status", "publish_at"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
