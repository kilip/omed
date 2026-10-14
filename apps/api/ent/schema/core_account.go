// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CoreAccount holds the schema definition for external OAuth accounts associated with a User.
type CoreAccount struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the CoreAccount table.
func (CoreAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Fields of the CoreAccount entity.
func (CoreAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.String("provider_id").NotEmpty(),
		field.String("account_id").NotEmpty(),
		field.Text("access_token").Optional(),
		field.Text("refresh_token").Optional(),
		field.Text("id_token").Optional(),
		field.Time("access_token_expires_at").Optional(),
		field.Time("refresh_token_expires_at").Optional(),
		field.String("scope").Optional(),
		field.String("password").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the CoreAccount entity.
func (CoreAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the CoreAccount entity.
func (CoreAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider_id", "account_id").Unique(),
		index.Fields("user_id"),
	}
}

