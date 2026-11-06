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

// AgentChatAttachment holds the schema definition for the AgentChatAttachment entity.
type AgentChatAttachment struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the AgentChatAttachment table.
func (AgentChatAttachment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("agent"),
	}
}

// Mixin of the AgentChatAttachment entity.
func (AgentChatAttachment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the AgentChatAttachment entity.
func (AgentChatAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("conversation_id"),
		field.Int64("message_id").Optional().Nillable(),
		field.String("object_key").NotEmpty(),
		field.String("bucket").NotEmpty(),
		field.String("mime").NotEmpty(),
		field.Int64("size").NonNegative(),
		field.String("file_name").NotEmpty(),
	}
}

// Edges of the AgentChatAttachment entity.
func (AgentChatAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("conversation", AgentChatConversation.Type).
			Field("conversation_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("message", AgentChatMessage.Type).
			Ref("attachments").
			Field("message_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the AgentChatAttachment entity.
func (AgentChatAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "conversation_id"),
		index.Fields("workspace_id", "message_id"),
	}
}
