package config

import (
	"context"
	"database/sql"
	"log"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the "pgx" driver

	"entgo.io/ent/dialect"
	"github.com/kilip/omed/api/ent"
)

func GetEntClient(cfg Config) *ent.Client {
	if cfg.DatabaseMigrateOnStart {
		if err := RunMigrations(context.Background(), cfg.DatabaseUrl); err != nil {
			log.Fatalf("failed to apply database migrations: %v", err)
		}
	}

	db, err := sql.Open("pgx", cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("Error while connecting to database %v", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))

	if cfg.DatabaseAutoMigrate {
		ctx := context.Background()
		for _, s := range []string{"core", "fin", "cms"} {
			if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+s); err != nil {
				log.Fatalf("failed creating schema %s: %v", s, err)
			}
		}
		if err := client.Schema.Create(ctx); err != nil {
			log.Fatalf("failed to creating schema resources: %v", err)
		}
	}

	return client
}
