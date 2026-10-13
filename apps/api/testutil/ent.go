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

func createTestDB() *ent.Client {
	// Open an in-memory SQLite client with automatic schema migration
	db, err := sql.Open("sqlite", "file:ent?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Can't create test db client: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)

	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}
	// config.ConfigureDBClient(client)
	return client
}
