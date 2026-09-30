package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Posting struct {
	ent.Schema
}

func (Posting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance", Table: "posting"},
		entsql.WithComments(true),
		entsql.Checks(map[string]string{
			"debit_or_credit_exclusive": "debit_amount = 0 OR credit_amount = 0",
			"amounts_non_negative":      "debit_amount >= 0 AND credit_amount >= 0",
		}),
	}
}

func (Posting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		WorkspaceMixin{},
	}
}

func (Posting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("entryId", uuid.UUID{}),
		field.UUID("accountId", uuid.UUID{}),
		field.String("currency").MaxLen(3),
		decimalField("debitAmount"),
		decimalField("creditAmount"),
		field.String("base_currency").MaxLen(3),
		decimalField("baseDebitAmount"),
		decimalField("baseCreditAmount"),
		decimalField("exchangeRate"),
		field.String("memo").Optional().Nillable(),
	}
}

func (Posting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entry", Entry.Type).
			Ref("postings").
			Field("entryId").
			Unique().
			Required(),
		edge.To("account", Account.Type).
			Field("accountId").
			Unique().
			Required(),
	}
}
