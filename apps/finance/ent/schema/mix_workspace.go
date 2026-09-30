package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

type WorkspaceMixin struct {
	mixin.Schema
}

func (WorkspaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("workspace_id", uuid.UUID{}),
	}
}

func (WorkspaceMixin) Edges() []ent.Edge {
	return nil
}

func (WorkspaceMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id"),
	}
}
