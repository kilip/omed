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

// Account holds the schema definition for the Account entity (Chart of Accounts).
type Account struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Account table.
func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the Account entity.
func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the Account entity.
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("workspace_id").Immutable(),
		field.String("code").NotEmpty(),
		field.String("name").NotEmpty(),
		field.Enum("type").Values("ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"),
		field.Enum("normal_balance").Values("DEBIT", "CREDIT"),
		field.Int64("parent_id").Optional().Nillable(),
		field.String("description").Optional(),
		field.Bool("is_active").Default(true),
		field.Bool("is_postable").Default(true),
	}
}

// Edges of the Account entity.
func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("workspace", Workspace.Type).
			Field("workspace_id").
			Unique().
			Required().
			Immutable(),
		edge.To("children", Account.Type).
			From("parent").
			Field("parent_id").
			Unique(),
	}
}

// Indexes of the Account entity.
func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "code").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("workspace_id", "parent_id"),
		index.Fields("workspace_id", "deleted_at"),
		index.Fields("workspace_id", "is_postable"),
	}
}
