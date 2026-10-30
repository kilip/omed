// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/kilip/omed/api/internal/platform/llmx"
)

// AgentProvider holds the schema definition for the AgentProvider entity.
type AgentProvider struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the AgentProvider table.
func (AgentProvider) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("agent"),
	}
}

// Mixin of the AgentProvider entity.
func (AgentProvider) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the AgentProvider entity.
func (AgentProvider) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.Enum("kind").Values("openai_compat", "anthropic", "gemini"),
		field.String("base_url").Optional(),
		field.String("model").NotEmpty(),
		field.String("api_key_enc").NotEmpty().Sensitive(),
		field.String("key_hint").Default(""),
		field.Bool("is_default").Default(false),
		field.JSON("capabilities", &llmx.Capabilities{}).Optional(),
	}
}

// Edges of the AgentProvider entity.
func (AgentProvider) Edges() []ent.Edge {
	return nil
}

// Indexes of the AgentProvider entity.
func (AgentProvider) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "name").Unique(),
		index.Fields("workspace_id").Unique().Annotations(entsql.IndexWhere("is_default")),
	}
}
