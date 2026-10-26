// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
)

// Transaction holds the schema definition for the Transaction entity.
//
// A transaction is the business event that caused money to move (an invoice
// paid, a salary received, an adjustment). It owns one or more journal
// entries, which in turn own the debit/credit postings.
type Transaction struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Transaction table.
func (Transaction) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the Transaction entity.
func (Transaction) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the Transaction entity.
func (Transaction) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("number").NotEmpty(),
		field.Time("transaction_date"),
		field.String("description").Optional(),
		field.String("reference").Optional(),
		field.String("currency").MinLen(3).MaxLen(3).Default("IDR"),
		field.Other("exchange_rate", decimal.Decimal{}).Default(decimal.NewFromInt(1)).SchemaType(map[string]string{
			dialect.Postgres: "numeric(20,8)",
			dialect.SQLite:   "numeric(20,8)",
		}),
		field.Enum("source").Values("MANUAL", "IMPORT", "SYSTEM").Default("MANUAL"),
	}
}

// Edges of the Transaction entity.
func (Transaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entries", Entry.Type),
	}
}

// Indexes of the Transaction entity.
func (Transaction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "number").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "transaction_date"),
		index.Fields("workspace_id", "reference"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
