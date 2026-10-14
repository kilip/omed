// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// AuditMixin provides audit fields and edges (created_by, created_at, updated_by, updated_at).
type AuditMixin struct {
	mixin.Schema
}

// Audit is an alias for AuditMixin.
type Audit = AuditMixin

// Fields of the AuditMixin.
func (AuditMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("created_by"),
		field.Int64("updated_by"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the AuditMixin.
func (AuditMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("creator", User.Type).
			Field("created_by").
			Unique().
			Required(),
		edge.To("updater", User.Type).
			Field("updated_by").
			Unique().
			Required(),
	}
}
