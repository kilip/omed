package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type LedgerPeriod struct {
	ent.Schema
}

func (LedgerPeriod) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance", Table: "ledger_period"},
		entsql.WithComments(true),
	}
}

func (LedgerPeriod) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		AuditMixins{},
		WorkspaceMixin{},
	}
}

func (LedgerPeriod) Fields() []ent.Field {
	return []ent.Field{
		field.Time("start_date"),
		field.Time("end_date"),
		field.Enum("status").Values("open", "closed", "locked").Default("open"),
	}
}

func (LedgerPeriod) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entries", Entry.Type),
	}
}
