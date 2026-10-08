package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

type AuditMixins struct {
	mixin.Schema
}

func (AuditMixins) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("createdBy", uuid.UUID{}),
		field.Time("createdAt").Immutable().Default(time.Now),
		field.UUID("updatedBy", uuid.UUID{}),
		field.Time("updatedAt").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (AuditMixins) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("creator", User.Type).
			Field("createdBy").
			Required().
			Unique(),
		edge.To("updater", User.Type).
			Field("updatedBy").
			Required().
			Unique(),
	}
}
