// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FinSetting holds the schema definition for the FinSetting entity (workspace financial settings).
type FinSetting struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the FinSetting table.
func (FinSetting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the FinSetting entity.
func (FinSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the FinSetting entity.
func (FinSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("fx_gain_account_id").Optional().Nillable(),
		field.Int64("fx_loss_account_id").Optional().Nillable(),
		field.Int64("fx_unrealized_gain_account_id").Optional().Nillable(),
		field.Int64("fx_unrealized_loss_account_id").Optional().Nillable(),
		field.Int64("fx_rounding_account_id").Optional().Nillable(),
	}
}

// Edges of the FinSetting entity.
func (FinSetting) Edges() []ent.Edge {
	return nil
}

// Indexes of the FinSetting entity.
func (FinSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id").Unique(),
	}
}
