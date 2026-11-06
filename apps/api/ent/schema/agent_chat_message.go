// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentChatMessage holds the schema definition for the AgentChatMessage entity.
type AgentChatMessage struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the AgentChatMessage table.
func (AgentChatMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("agent"),
	}
}

// Mixin of the AgentChatMessage entity.
func (AgentChatMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the AgentChatMessage entity.
func (AgentChatMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("conversation_id"),
		field.Enum("role").Values("user", "assistant", "system", "tool"),
		field.Text("content").Default(""),
		field.JSON("parts", json.RawMessage{}).Optional(),
		field.Enum("draft_status").Values("none", "proposed", "confirmed", "discarded").Default("none"),
		field.Int64("confirmed_transaction_id").Optional().Nillable(),
	}
}

// Edges of the AgentChatMessage entity.
func (AgentChatMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", AgentChatConversation.Type).
			Ref("messages").
			Field("conversation_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("attachments", AgentChatAttachment.Type),
	}
}

// Indexes of the AgentChatMessage entity.
func (AgentChatMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "conversation_id"),
	}
}
