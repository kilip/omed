// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Currency holds the schema definition for the Currency entity.
//
// Currency is global reference data (ISO 4217), not scoped per workspace.
type Currency struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Currency table.
func (Currency) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Mixin of the Currency entity.
func (Currency) Mixin() []ent.Mixin {
	return []ent.Mixin{
		AuditMixin{},
	}
}

// Fields of the Currency entity.
func (Currency) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("code").NotEmpty().Unique().MinLen(3).MaxLen(3),
		field.String("name").NotEmpty(),
		field.String("symbol").NotEmpty(),
		field.Int("decimals").NonNegative(),
	}
}

// Edges of the Currency entity.
func (Currency) Edges() []ent.Edge {
	return nil
}

// Indexes of the Currency entity.
func (Currency) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
	}
}
