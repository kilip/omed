// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// PublishMixin provides publication status and scheduling fields (status, publish_at).
type PublishMixin struct {
	mixin.Schema
}

// Publish is an alias for PublishMixin.
type Publish = PublishMixin

// Fields of the PublishMixin.
func (PublishMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").Values("DRAFT", "PUBLISHED").Default("DRAFT"),
		field.Time("publish_at").Optional().Nillable(),
	}
}
