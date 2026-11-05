// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CmsSetting holds the schema definition for the CmsSetting entity.
type CmsSetting struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CmsSetting table.
func (CmsSetting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("cms"),
	}
}

// Mixin of the CmsSetting entity.
func (CmsSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the CmsSetting entity.
func (CmsSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("default_locale").Default("en"),
		field.Strings("supported_locales").Default([]string{"en", "id"}),
	}
}

// Edges of the CmsSetting entity.
func (CmsSetting) Edges() []ent.Edge {
	return nil
}

// Indexes of the CmsSetting entity.
func (CmsSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id").Unique(),
	}
}
