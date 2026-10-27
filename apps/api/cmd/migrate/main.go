// Package main provides the Atlas multi-schema migration diff generator and drift checker for Ent schemas.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	checkOnly := flag.Bool("check", false, "Check if Ent schemas have un-migrated changes without writing files")
	dirPath := flag.String("dir", "migrations", "Path to the migration directory")
	schemaPath := flag.String("schema", "./ent/schema", "Path to the Ent schema directory")
	flag.Parse()

	name := "changes"
	if flag.NArg() > 0 && strings.TrimSpace(flag.Arg(0)) != "" {
		name = strings.TrimSpace(flag.Arg(0))
	}

	ctx := context.Background()
	devURL := resolveDevURL()

	if err := ensureDevDatabase(ctx, devURL); err != nil {
		log.Fatalf("migrate: failed ensuring dev database: %v", err)
	}

	if err := os.MkdirAll(*dirPath, 0o755); err != nil {
		log.Fatalf("migrate: failed creating migrations directory: %v", err)
	}

	dir, err := migrate.NewLocalDir(*dirPath)
	if err != nil {
		log.Fatalf("migrate: failed opening migration directory %q: %v", *dirPath, err)
	}

	if err := migrate.Validate(dir); err != nil {
		log.Fatalf("migrate: migration directory validation failed: %v", err)
	}

	devDB, err := sql.Open("pgx", devURL)
	if err != nil {
		log.Fatalf("migrate: failed connecting to dev database: %v", err)
	}
	defer func() { _ = devDB.Close() }()

	if err := cleanSchemas(ctx, devDB, []string{"core", "fin", "cms"}); err != nil {
		log.Fatalf("migrate: failed cleaning dev schemas before replay: %v", err)
	}
	defer func() {
		_ = cleanSchemas(ctx, devDB, []string{"core", "fin", "cms"})
	}()

	atDriver, err := postgres.Open(devDB)
	if err != nil {
		log.Fatalf("migrate: failed opening atlas postgres driver: %v", err)
	}

	graph, err := entc.LoadGraph(*schemaPath, &gen.Config{})
	if err != nil {
		log.Fatalf("migrate: failed loading ent graph: %v", err)
	}

	tables, err := graph.Tables()
	if err != nil {
		log.Fatalf("migrate: failed computing ent tables: %v", err)
	}

	entDrv := entsql.OpenDB(dialect.Postgres, devDB)
	entMigrate, err := entschema.NewMigrate(entDrv)
	if err != nil {
		log.Fatalf("migrate: failed creating ent migrate instance: %v", err)
	}

	rawRealm, err := entMigrate.StateReader(tables...).ReadState(ctx)
	if err != nil {
		log.Fatalf("migrate: failed reading ent realm state: %v", err)
	}

	normalizer, ok := atDriver.(schema.Normalizer)
	if !ok {
		log.Fatalf("migrate: postgres driver does not implement schema.Normalizer")
	}

	desiredRealm, err := normalizer.NormalizeRealm(ctx, rawRealm)
	if err != nil {
		log.Fatalf("migrate: failed normalizing desired realm: %v", err)
	}

	planner := migrate.NewPlanner(
		atDriver,
		dir,
		migrate.PlanFormat(migrate.DefaultFormatter),
		migrate.PlanWithExclude("public", "public.*"),
	)

	plan, err := planner.Plan(ctx, name, migrate.StateReaderFunc(func(context.Context) (*schema.Realm, error) {
		return desiredRealm, nil
	}))
	if errors.Is(err, migrate.ErrNoPlan) {
		fmt.Println("migrate: schema is up to date, no changes detected")
		return
	}
	if err != nil {
		log.Fatalf("migrate: failed planning migration: %v", err)
	}

	if *checkOnly {
		log.Fatalf("migrate: detected %d un-migrated schema change(s); run 'bun run db:diff <name>' to generate migration files", len(plan.Changes))
	}

	if err := planner.WritePlan(plan); err != nil {
		log.Fatalf("migrate: failed writing migration plan: %v", err)
	}

	fmt.Printf("migrate: generated migration %q with %d statement(s)\n", plan.Name, len(plan.Changes))
}

func resolveDevURL() string {
	if devURL := strings.TrimSpace(os.Getenv("OMED_DB_DEV_URL")); devURL != "" {
		return devURL
	}
	if dbURL := strings.TrimSpace(os.Getenv("OMED_DB_URL")); dbURL != "" {
		u, err := url.Parse(dbURL)
		if err == nil {
			u.Path = "/omed_dev"
			q := u.Query()
			q.Del("search_path")
			u.RawQuery = q.Encode()
			return u.String()
		}
	}
	return "postgres://omed:omed@localhost:5432/omed_dev?sslmode=disable"
}

func ensureDevDatabase(ctx context.Context, devURL string) error {
	u, err := url.Parse(devURL)
	if err != nil {
		return fmt.Errorf("parse dev URL: %w", err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		dbName = "omed_dev"
	}

	adminURL := *u
	adminURL.Path = "/postgres"
	db, err := sql.Open("pgx", adminURL.String())
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		// Fallback to /omed if /postgres is not accessible
		adminURL.Path = "/omed"
		_ = db.Close()
		db, err = sql.Open("pgx", adminURL.String())
		if err != nil {
			return fmt.Errorf("open fallback admin db: %w", err)
		}
		defer func() { _ = db.Close() }()
	}

	var exists int
	err = db.QueryRowContext(ctx, "SELECT 1 FROM pg_database WHERE datname = $1", dbName).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = db.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, strings.ReplaceAll(dbName, `"`, `""`)))
		if err != nil {
			return fmt.Errorf("create database %q: %w", dbName, err)
		}
		return nil
	}
	return err
}

func cleanSchemas(ctx context.Context, db *sql.DB, schemas []string) error {
	for _, s := range schemas {
		quoted := fmt.Sprintf(`"%s"`, strings.ReplaceAll(s, `"`, `""`))
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoted)); err != nil {
			return err
		}
	}
	return nil
}

