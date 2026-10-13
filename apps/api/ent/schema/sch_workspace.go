// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Workspace holds the schema definition for the Workspace entity.
type Workspace struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the Workspace table.
func (Workspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Fields of the Workspace entity.
func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.String("description").Optional(),
		field.Int64("owner_id"),
		field.Int64("created_by_id").Optional().Nillable(),
		field.Int64("updated_by_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Workspace entity.
func (Workspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owner", User.Type).
			Field("owner_id").
			Unique().
			Required(),
		edge.To("creator", User.Type).
			Field("created_by_id").
			Unique(),
		edge.To("updater", User.Type).
			Field("updated_by_id").
			Unique(),
	}
}

