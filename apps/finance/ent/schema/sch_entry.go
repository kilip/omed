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

type Entry struct {
	ent.Schema
}

func (Entry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Schema: "finance", Table: "entry"},
		entsql.WithComments(true),
	}
}

func (Entry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
		AuditMixins{},
		WorkspaceMixin{},
	}
}

func (Entry) Fields() []ent.Field {
	return []ent.Field{
		field.Time("entry_date"),
		field.Enum("entry_type").
			Values("normal", "opening_balance", "adjustment", "closing", "fx_adjustment").
			Default("normal"),
		field.String("description").Optional(),
		field.String("reference").Optional().Nillable(),
		field.UUID("period_id", uuid.UUID{}),
	}
}

func (Entry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("period", Period.Type).
			Ref("entries").
			Field("period_id").
			Unique().
			Required(),
		/*
			edge.To("postings", Posting.Type),
			edge.To("attachments", Attachment.Type),
		*/
	}
}

func (Entry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "entry_date"),
	}
}
