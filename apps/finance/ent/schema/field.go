package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/shopspring/decimal"
)

func decimalField(name string) ent.Field {
	return field.Other(name, decimal.Decimal{}).
		SchemaType(map[string]string{
			dialect.Postgres: "numeric(20,8)",
			dialect.SQLite:   "numeric",
		}).
		Default(decimal.Zero)
}
