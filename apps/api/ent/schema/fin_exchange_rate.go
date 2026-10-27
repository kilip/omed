// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/shopspring/decimal"
)

// ExchangeRate holds the schema definition for the ExchangeRate entity.
//
// ExchangeRate stores currency conversion rates per workspace.
type ExchangeRate struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the ExchangeRate table.
func (ExchangeRate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("fin"),
	}
}

// Mixin of the ExchangeRate entity.
func (ExchangeRate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the ExchangeRate entity.
func (ExchangeRate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("from_currency").MinLen(3).MaxLen(3),
		field.String("to_currency").MinLen(3).MaxLen(3),
		field.Other("rate", decimal.Decimal{}).SchemaType(map[string]string{
			dialect.Postgres: "numeric(20,8)",
			dialect.SQLite:   "numeric(20,8)",
		}),
		field.Time("effective_date"),
		field.Enum("source").Values("MANUAL", "IMPORT").Default("MANUAL"),
	}
}

// Edges of the ExchangeRate entity.
func (ExchangeRate) Edges() []ent.Edge {
	return nil
}

// Indexes of the ExchangeRate entity.
func (ExchangeRate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "from_currency", "to_currency", "effective_date").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "deleted_at"),
	}
}
