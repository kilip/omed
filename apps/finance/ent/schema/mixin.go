package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

type TimeMixin struct {
	mixin.Schema
}

func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("createdAt").Default(time.Now).Immutable(),
		field.Time("updatedAt").Default(time.Now).UpdateDefault(time.Now),
	}
}

type IDV7Mixin struct {
	mixin.Schema
}

func (IDV7Mixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(func() uuid.UUID {
			id, err := uuid.NewV7()
			if err != nil {
				panic(err) // Fallback behavior for generation failure
			}
			return id
		}),
	}
}

type WorkspaceMixin struct {
	mixin.Schema
}

func (WorkspaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("workspaceId", uuid.UUID{}),
	}
}

func (WorkspaceMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspaceId"),
	}
}

func (WorkspaceMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("workspace", Workspace.Type).Required().Unique().Field("workspaceId"),
	}
}

type AuditMixin struct {
	mixin.Schema
}

func (AuditMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("createdBy", uuid.UUID{}),
		field.UUID("updatedBy", uuid.UUID{}),
	}
}

func (AuditMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("creator", User.Type).Required().Unique().Field("createdBy"),
		edge.To("updater", User.Type).Required().Unique().Field("updatedBy"),
	}
}
