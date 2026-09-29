package config

import (
	"context"
	"log"

	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the "pgx" driver
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/account"
	"github.com/kilip/omed/finance/ent/entry"
	"github.com/kilip/omed/finance/ent/exchangerate"
	"github.com/kilip/omed/finance/ent/hook"
	"github.com/kilip/omed/finance/ent/ledgerperiod"
	"github.com/kilip/omed/finance/ent/posting"
	"github.com/kilip/omed/finance/internal/shared"
)

func WorkspaceInterceptor() ent.Interceptor {
	return ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			user := shared.UserFromContext(ctx)
			if user.WorkspaceID != uuid.Nil {
				switch query := q.(type) {
				case *ent.AccountQuery:
					query.Where(account.WorkspaceIDEQ(user.WorkspaceID))
				case *ent.EntryQuery:
					query.Where(entry.WorkspaceIDEQ(user.WorkspaceID))
				case *ent.ExchangeRateQuery:
					query.Where(exchangerate.WorkspaceIDEQ(user.WorkspaceID))
				case *ent.LedgerPeriodQuery:
					query.Where(ledgerperiod.WorkspaceIDEQ(user.WorkspaceID))
				case *ent.PostingQuery:
					query.Where(posting.WorkspaceIDEQ(user.WorkspaceID))
				}
			}
			return next.Query(ctx, q)
		})
	})
}

func WorkspaceHook() ent.Hook {
	hk := func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			user := shared.UserFromContext(ctx)
			switch m.Op() {
			case ent.OpCreate:
				if setter, ok := m.(interface{ SetWorkspaceID(uuid.UUID) }); ok {
					setter.SetWorkspaceID(user.WorkspaceID)
				}
				if setter, ok := m.(interface{ SetCreatedBy(uuid.UUID) }); ok {
					setter.SetCreatedBy(user.ID)
				}
				if setter, ok := m.(interface{ SetCreatedByName(string) }); ok {
					setter.SetCreatedByName(user.Name)
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
				if setter, ok := m.(interface{ SetUpdatedByName(string) }); ok {
					setter.SetUpdatedByName(user.Name)
				}
			case ent.OpUpdate, ent.OpUpdateOne:
				if f, ok := m.(interface {
					WhereP(...func(*entsql.Selector))
				}); ok {
					f.WhereP(entsql.FieldEQ("workspace_id", user.WorkspaceID))
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpDelete, ent.OpDeleteOne:
				if f, ok := m.(interface {
					WhereP(...func(*entsql.Selector))
				}); ok {
					f.WhereP(entsql.FieldEQ("workspace_id", user.WorkspaceID))
				}
			}

			return next.Mutate(ctx, m)
		})
	}
	return hook.On(hk, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne|ent.OpDelete|ent.OpDeleteOne)
}

func ConfigureDBClient(client *ent.Client) {
	client.Use(WorkspaceHook())
	client.Intercept(WorkspaceInterceptor())
}

func GetEntClient(cfg Config) *ent.Client {
	db, err := sql.Open("pgx", cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("Error while connecting to database %v", err)
	}

	// 1. Ensure the database container schema physically exists before migration runs
	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS finance;")
	if err != nil {
		log.Fatalf("Failed to guarantee schema container existence: %v", err)
	}
	// set current schema
	_, err = db.Exec("SET search_path TO finance;")
	if err != nil {
		log.Fatalf("Can't set search_path: %v", err)
	}

	schemaCfg := ent.SchemaConfig{
		Account:      "finance",
		Entry:        "finance",
		ExchangeRate: "finance",
		LedgerPeriod: "finance",
		Posting:      "finance",
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv), ent.AlternateSchema(schemaCfg))

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	ConfigureDBClient(client)

	return client
}
