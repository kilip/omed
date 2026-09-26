package testutil

import (
	"context"
	"log"

	"github.com/kilip/omed/finance/ent"
	_ "github.com/mattn/go-sqlite3"
)

func createTestDB() *ent.Client {
	// Open an in-memory SQLite client with automatic schema migration
	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("Can't create test db client: %v", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	return client
}
