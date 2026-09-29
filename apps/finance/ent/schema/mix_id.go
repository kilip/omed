package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/shared"
)

type IDV7Mixin struct {
	mixin.Schema
}

func (IDV7Mixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(shared.GenerateID).Immutable(),
	}
}
