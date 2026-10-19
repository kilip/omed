// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Entry holds the schema definition for the Entry entity (Journal Entry).
//
// An entry groups postings that must balance (total debit equals total credit).
// The balance invariant is enforced by the service layer when an entry moves
// from DRAFT to POSTED. A POSTED entry is immutable: corrections are made with
// a reversing entry (reversal_of_id), never by editing or deleting.
type Entry struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Entry table.
func (Entry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the Entry entity.
func (Entry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the Entry entity.
func (Entry) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("transaction_id").Immutable(),
		field.Int64("period_id"),
		field.String("number").NotEmpty(),
		field.Time("entry_date"),
		field.String("description").Optional(),
		field.Enum("status").Values("DRAFT", "POSTED", "VOID").Default("DRAFT"),
		field.Time("posted_at").Optional().Nillable(),
		field.Int64("posted_by").Optional().Nillable(),
		field.Int64("reversal_of_id").Optional().Nillable(),
	}
}

// Edges of the Entry entity.
func (Entry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("transaction", Transaction.Type).
			Ref("entries").
			Field("transaction_id").
			Unique().
			Required().
			Immutable(),
		edge.To("period", Period.Type).
			Field("period_id").
			Unique().
			Required(),
		edge.To("poster", User.Type).
			Field("posted_by").
			Unique(),
		edge.To("reversals", Entry.Type).
			From("reversal_of").
			Field("reversal_of_id").
			Unique(),
		edge.To("postings", Posting.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the Entry entity.
func (Entry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "number").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "transaction_id"),
		index.Fields("workspace_id", "period_id", "status"),
		index.Fields("workspace_id", "entry_date"),
		index.Fields("workspace_id", "status"),
		index.Fields("workspace_id", "reversal_of_id"),
		index.Fields("workspace_id", "deleted_at"),
	}
}
