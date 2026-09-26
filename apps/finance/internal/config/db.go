package config

import (
	"context"
	"log"
	"strings"

	"database/sql"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the "pgx" driver
	"github.com/kilip/omed/finance/ent"
)

func runMigration(ctx context.Context, client *ent.Client) error {
	targetSchema := "finance"

	return client.Schema.Create(
		ctx,
		// 1. Hook cleanly into the runtime execution engine
		schema.WithApplyHook(func(next schema.Applier) schema.Applier {
			// 2. Use schema.ApplyFunc adapter to construct the return type
			return schema.ApplyFunc(func(ctx context.Context, conn dialect.ExecQuerier, plan *migrate.Plan) error {
				for _, change := range plan.Changes {
					// 3. Force rewrite all structural modifiers targeting "public"
					if strings.Contains(change.Cmd, "public.") {
						change.Cmd = strings.ReplaceAll(change.Cmd, "public.", targetSchema+".")
					}
					if strings.Contains(change.Cmd, "\"public\".") {
						change.Cmd = strings.ReplaceAll(change.Cmd, "\"public\".", "\""+targetSchema+"\".")
					}
				}
				return next.Apply(ctx, conn, plan)
			})
		}),
	)
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
	_, err = db.Exec("SET search_path TO finance, public;")
	if err != nil {
		log.Fatalf("Can't set search_path: %v", err)
	}

	schemaCfg := ent.SchemaConfig{
		User:      "finance",
		Account:   "finance",
		Workspace: "finance",
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv), ent.AlternateSchema(schemaCfg))

	/**
	if err := runMigration(context.Background(), client); err != nil {
		log.Fatalf("Error running migration: %v", err)
	}*/

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	return client
}
