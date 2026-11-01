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

// PersonalAccessToken holds the schema definition for personal access tokens.
type PersonalAccessToken struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the PersonalAccessToken table.
func (PersonalAccessToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("core"),
	}
}

// Fields of the PersonalAccessToken entity.
func (PersonalAccessToken) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.String("name").NotEmpty(),
		field.String("token_hash").NotEmpty().Unique(),
		field.String("prefix").NotEmpty(),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the PersonalAccessToken entity.
func (PersonalAccessToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the PersonalAccessToken entity.
func (PersonalAccessToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}
