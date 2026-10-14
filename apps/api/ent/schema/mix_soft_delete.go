// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// SoftDeleteMixin provides soft delete fields and relations (deleted_at, deleted_by_id, deleter).
type SoftDeleteMixin struct {
	mixin.Schema
}

// SoftDelete is an alias for SoftDeleteMixin.
type SoftDelete = SoftDeleteMixin

// Fields of the SoftDeleteMixin.
func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted_at").Optional().Nillable(),
		field.Int64("deleted_by_id").Optional().Nillable(),
	}
}

// Edges of the SoftDeleteMixin.
func (SoftDeleteMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("deleter", User.Type).
			Field("deleted_by_id").
			Unique(),
	}
}
