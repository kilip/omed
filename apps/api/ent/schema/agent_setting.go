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

// AgentSetting holds the schema definition for the AgentSetting entity (workspace agent settings).
type AgentSetting struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the AgentSetting table.
func (AgentSetting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("agent"),
	}
}

// Mixin of the AgentSetting entity.
func (AgentSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the AgentSetting entity.
func (AgentSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Bool("financial_enabled").Default(false),
		field.Int64("financial_provider_id").Optional().Nillable(),
	}
}

// Edges of the AgentSetting entity.
func (AgentSetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("financial_provider", AgentProvider.Type).
			Field("financial_provider_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

// Indexes of the AgentSetting entity.
func (AgentSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id").Unique(),
	}
}
