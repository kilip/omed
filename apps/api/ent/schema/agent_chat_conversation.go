// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentChatConversation holds the schema definition for the AgentChatConversation entity.
type AgentChatConversation struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the AgentChatConversation table.
func (AgentChatConversation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("agent"),
	}
}

// Mixin of the AgentChatConversation entity.
func (AgentChatConversation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
		SoftDeleteMixin{},
	}
}

// Fields of the AgentChatConversation entity.
func (AgentChatConversation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("title").Optional(),
	}
}

// Edges of the AgentChatConversation entity.
func (AgentChatConversation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("messages", AgentChatMessage.Type),
	}
}

// Indexes of the AgentChatConversation entity.
func (AgentChatConversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "deleted_at"),
		index.Fields("workspace_id", "created_by"),
	}
}
