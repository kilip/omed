package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Period struct {
	ent.Schema
}

func (Period) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance", Table: "period"},
		entsql.WithComments(true),
	}
}

func (Period) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		AuditMixins{},
		WorkspaceMixin{},
	}
}

func (Period) Fields() []ent.Field {
	return []ent.Field{
		field.Time("start_date"),
		field.Time("end_date"),
		field.Enum("status").Values("open", "closed", "locked").Default("open"),
	}
}

func (Period) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entries", Entry.Type),
	}
}
