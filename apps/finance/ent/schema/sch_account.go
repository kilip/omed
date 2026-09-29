package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Account struct {
	ent.Schema
}

func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance", Table: "account"},
	}
}

func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		AuditMixins{},
		WorkspaceMixin{},
	}
}

func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("description").Optional().Nillable(),
		field.Enum("type").Values("asset", "liability", "equity", "revenue", "expense"),
		field.String("currency").MaxLen(3),
		field.Enum("status").Values("active", "archived").Default("active"),
		field.UUID("parentId", uuid.UUID{}).Optional().Nillable().StructTag("parentId"),
	}
}

func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Account.Type).
			From("parent").
			Field("parentId").
			Unique(),
	}
}

func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "code").Unique(),
	}
}
