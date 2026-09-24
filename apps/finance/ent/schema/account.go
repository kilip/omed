package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Account holds the schema definition for the Account entity.
type Account struct {
	ent.Schema
}

func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance"},
	}
}

func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		TimeMixin{},
		WorkspaceMixin{},
		AuditMixin{},
	}
}

// Fields of the Account.
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("parentId", uuid.UUID{}).Optional().Nillable(),
		field.String("code").NotEmpty(),
		field.String("name").NotEmpty(),
		field.Enum("type").Values("asset", "liability", "equity", "income", "expense"),
	}
}

// Edges of the Account.
func (Account) Edges() []ent.Edge {
	return nil
}
