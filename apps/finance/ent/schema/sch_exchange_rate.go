package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ExchangeRate struct {
	ent.Schema
}

func (ExchangeRate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance", Table: "exchange_rate"},
		entsql.WithComments(true),
	}
}

func (ExchangeRate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		WorkspaceMixin{},
	}
}

func (ExchangeRate) Fields() []ent.Field {
	return []ent.Field{
		field.String("fromCurrency").MaxLen(3),
		field.String("toCurrency").MaxLen(3),
		decimalField("rate"),
		field.Time("rateDate"),
		field.String("source").Optional().Nillable(),
	}
}

func (ExchangeRate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "fromCurrency", "toCurrency", "rateDate").Unique(),
	}
}
