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
)

// Posting holds the schema definition for the Posting entity.
//
// A posting is a single debit or credit line against an account. The amount is
// always positive; the direction decides which side of the ledger it lands on.
// Postings have no soft delete: they live and die with their entry
// (only DRAFT entries can be removed, which cascades to their postings).
type Posting struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Posting table.
func (Posting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the Posting entity.
func (Posting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the Posting entity.
func (Posting) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("entry_id").Immutable(),
		field.Int64("account_id"),
		field.Enum("direction").Values("DEBIT", "CREDIT"),
		field.Float("amount").Positive().SchemaType(map[string]string{dialect.Postgres: "numeric(20,4)"}),
		field.Float("base_amount").Positive().SchemaType(map[string]string{dialect.Postgres: "numeric(20,4)"}),
		field.Int("position").NonNegative().Default(0),
		field.String("description").Optional(),
	}
}

// Edges of the Posting entity.
func (Posting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entry", Entry.Type).
			Ref("postings").
			Field("entry_id").
			Unique().
			Required().
			Immutable(),
		edge.To("account", Account.Type).
			Field("account_id").
			Unique().
			Required(),
	}
}

// Indexes of the Posting entity.
func (Posting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("entry_id", "position").
			Unique(),
		index.Fields("workspace_id", "entry_id"),
		index.Fields("workspace_id", "account_id"),
		index.Fields("workspace_id", "account_id", "direction"),
	}
}
