// Package schema defines the Ent schema specifications for database entities.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LoginOTP holds the schema definition for passwordless login one-time passwords.
type LoginOTP struct {
	ent.Schema
}

// Annotations returns additional schema annotations for the LoginOTP table.
func (LoginOTP) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Schema("core"),
		entsql.Annotation{Table: "login_otps"},
	}
}

// Fields of the LoginOTP entity.
func (LoginOTP) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("email").NotEmpty(),
		field.String("code_hash").NotEmpty(),
		field.Time("expires_at"),
		field.Int("attempts").Default(0),
		field.Time("consumed_at").Optional().Nillable(),
		field.String("ip_address").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the LoginOTP entity.
func (LoginOTP) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email", "created_at"),
		index.Fields("expires_at"),
	}
}
