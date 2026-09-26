package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type AuditMixins struct {
	mixin.Schema
}

func (AuditMixins) Fields() []ent.Field {
	return []ent.Field{
		field.Time("createdAt").Immutable().Default(time.Now),
		field.Time("updatedAt").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (AuditMixins) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("createdBy", User.Type),
		edge.From("updatedBy", User.Type),
	}
}
