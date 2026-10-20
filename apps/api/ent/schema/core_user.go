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

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the User table.
func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("omed"),
	}
}

// Fields of the User entity.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.String("email").NotEmpty().Unique(),
		field.String("avatar").Optional(),
		field.String("locale").Default("en").NotEmpty(),
		field.String("currency").Default("USD").MinLen(3).MaxLen(3),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("accounts", UserAccount.Type).Ref("user"),
		edge.From("sessions", Session.Type).Ref("user"),
	}
}
