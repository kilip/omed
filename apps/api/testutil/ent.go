package testutil

import (
	"context"
	"database/sql"
	"log"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/api/ent"
	_ "modernc.org/sqlite"
)

// CreateTestDB creates an in-memory SQLite client with automatic schema migration.
// If dsn is provided, the first dsn string is used; otherwise, a default in-memory DSN is used.
func CreateTestDB(dsn ...string) *ent.Client {
	connStr := "file:ent?mode=memory&cache=shared&_fk=1&_busy_timeout=5000"
	if len(dsn) > 0 && dsn[0] != "" {
		connStr = dsn[0]
	}

	// Open an in-memory SQLite client with automatic schema migration
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		log.Fatalf("Can't create test db client: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)

	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to create schema resources: %v", err)
	}
	// config.ConfigureDBClient(client)
	return client
}
