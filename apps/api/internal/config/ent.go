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
	// Configure which table maps to which Postgres schema
	schemas := ent.AlternateSchema(ent.SchemaConfig{
		User:                "public",
		Session:             "public",
		UserAccount:         "public",
		Workspace:           "public",
		WorkspaceInvitation: "public",
		WorkspaceMember:     "public",
		File:                "public",
		Account:             "public",
	})

	db, err := sql.Open("pgx", cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("Error while connecting to database %v", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv), schemas)

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	return client
}
