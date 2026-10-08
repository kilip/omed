package config

import (
	"context"
	"database/sql"
	"log"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/finance/ent/account"
	"github.com/kilip/omed/finance/ent/entry"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the "pgx" driver
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/hook"
	"github.com/kilip/omed/finance/internal/core"
)

func workspaceHook() ent.Hook {
	hk := func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			user := core.UserFromContext(ctx)
			_, scoped := m.(interface{ SetWorkspaceID(uuid.UUID) })

			switch m.Op() {
			case ent.OpCreate:
				if scoped {
					m.(interface{ SetWorkspaceID(uuid.UUID) }).SetWorkspaceID(user.WorkspaceID)
				}
				if setter, ok := m.(interface{ SetCreatedBy(uuid.UUID) }); ok {
					setter.SetCreatedBy(user.ID)
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpUpdate, ent.OpUpdateOne:
				if scoped {
					if f, ok := m.(interface {
						WhereP(...func(*entsql.Selector))
					}); ok {
						f.WhereP(entsql.FieldEQ("workspace_id", user.WorkspaceID))
					}
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpDelete, ent.OpDeleteOne:
				if scoped {
					if f, ok := m.(interface {
						WhereP(...func(*entsql.Selector))
					}); ok {
						f.WhereP(entsql.FieldEQ("workspace_id", user.WorkspaceID))
					}
				}
			}

			return next.Mutate(ctx, m)
		})

	}
	return hook.On(hk, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne|ent.OpDelete|ent.OpDeleteOne)
}

func workspaceInterceptor() ent.Interceptor {
	return ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			user := core.UserFromContext(ctx)
			if user.WorkspaceID != uuid.Nil {
				switch query := q.(type) {
				case *ent.AccountQuery:
					query.Where(account.WorkspaceIDEQ(user.WorkspaceID))
				case *ent.EntryQuery:
					query.Where(entry.WorkspaceIDEQ(user.WorkspaceID))
				}
			}
			return next.Query(ctx, q)
		})
	})
}

func ConfigureDBClient(client *ent.Client) {
	client.Use(workspaceHook())
	client.Intercept(workspaceInterceptor())
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
		Account:   "finance",
		Entry:     "finance",
		User:      "finance",
		Workspace: "finance",
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv), ent.AlternateSchema(schemaCfg))

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	ConfigureDBClient(client)

	return client
}
