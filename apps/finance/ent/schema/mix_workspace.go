package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

type WorkspaceMixin struct {
	mixin.Schema
}

func (WorkspaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("workspaceId", uuid.UUID{}),
	}
}

func (WorkspaceMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("workspace", Workspace.Type).Field("workspaceId").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
