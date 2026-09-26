package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Workspace holds the schema definition for the Workspace entity.
type Workspace struct {
	ent.Schema
}

func (Workspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Schema: "finance",
			Table:  "workspace",
		},
		entsql.WithComments(true),
	}
}

func (Workspace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
	}
}

func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.Time("syncedAt"),
	}
}
