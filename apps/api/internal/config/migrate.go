package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/kilip/omed/api/migrations"
)

const (
	revisionsSchema     = "atlas_schema_revisions"
	revisionsTable      = "atlas_schema_revisions"
	migrationLockID     = int64(784512963)
	defaultOperatorName = "omed-migrator"
)

// RunMigrations validates and applies all pending versioned SQL migrations from the embedded migrations directory.
func RunMigrations(ctx context.Context, databaseURL string) error {
	return RunMigrationsFromFS(ctx, databaseURL, migrations.FS)
}

// RunMigrationsFromFS validates and applies all pending versioned SQL migrations from the given filesystem.
func RunMigrationsFromFS(ctx context.Context, databaseURL string, mfs fs.FS) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("migrate: database URL is required")
	}

	dir, err := loadMemDir(mfs)
	if err != nil {
		return fmt.Errorf("migrate: loading embedded migrations: %w", err)
	}

	if err := migrate.Validate(dir); err != nil {
		return fmt.Errorf("migrate: validating migrations checksum: %w", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("migrate: opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("migrate: pinging database: %w", err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquiring database connection: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("migrate: acquiring advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()

	drv, err := postgres.Open(db)
	if err != nil {
		return fmt.Errorf("migrate: opening atlas postgres driver: %w", err)
	}

	rrw := &pgRevisionReadWriter{db: db}
	if err := rrw.init(ctx); err != nil {
		return fmt.Errorf("migrate: initializing revisions table: %w", err)
	}

	exec, err := migrate.NewExecutor(
		drv,
		dir,
		rrw,
		migrate.WithOperatorVersion(defaultOperatorName),
	)
	if err != nil {
		return fmt.Errorf("migrate: creating executor: %w", err)
	}

	if err := exec.ExecuteN(ctx, 0); err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		return fmt.Errorf("migrate: applying migrations: %w", err)
	}

	return nil
}

func loadMemDir(mfs fs.FS) (*migrate.MemDir, error) {
	entries, err := fs.ReadDir(mfs, ".")
	if err != nil {
		return nil, err
	}
	dir := &migrate.MemDir{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name != migrate.HashFileName && !strings.HasSuffix(name, ".sql") {
			continue
		}
		data, err := fs.ReadFile(mfs, name)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", name, err)
		}
		if err := dir.WriteFile(name, data); err != nil {
			return nil, fmt.Errorf("writing %q to memdir: %w", name, err)
		}
	}
	return dir, nil
}

type pgRevisionReadWriter struct {
	db *sql.DB
}

func (r *pgRevisionReadWriter) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{
		Schema: revisionsSchema,
		Name:   revisionsTable,
	}
}

func (r *pgRevisionReadWriter) init(ctx context.Context) error {
	const ddl = `
CREATE SCHEMA IF NOT EXISTS "atlas_schema_revisions";
CREATE TABLE IF NOT EXISTS "atlas_schema_revisions"."atlas_schema_revisions" (
  "version" character varying NOT NULL,
  "description" character varying NOT NULL,
  "type" bigint NOT NULL DEFAULT 2,
  "applied" bigint NOT NULL DEFAULT 0,
  "total" bigint NOT NULL DEFAULT 0,
  "executed_at" timestamptz NOT NULL,
  "execution_time" bigint NOT NULL,
  "error" text NULL,
  "error_stmt" text NULL,
  "hash" character varying NOT NULL,
  "partial_hashes" jsonb NULL,
  "operator_version" character varying NOT NULL,
  PRIMARY KEY ("version")
);`
	_, err := r.db.ExecContext(ctx, ddl)
	return err
}

func (r *pgRevisionReadWriter) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	const q = `
SELECT "version", "description", "type", "applied", "total", "executed_at", "execution_time",
       COALESCE("error", ''), COALESCE("error_stmt", ''), "hash", "partial_hashes", "operator_version"
FROM "atlas_schema_revisions"."atlas_schema_revisions"
ORDER BY "version" ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var revs []*migrate.Revision
	for rows.Next() {
		rev, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return revs, nil
}

func (r *pgRevisionReadWriter) ReadRevision(ctx context.Context, version string) (*migrate.Revision, error) {
	const q = `
SELECT "version", "description", "type", "applied", "total", "executed_at", "execution_time",
       COALESCE("error", ''), COALESCE("error_stmt", ''), "hash", "partial_hashes", "operator_version"
FROM "atlas_schema_revisions"."atlas_schema_revisions"
WHERE "version" = $1`

	row := r.db.QueryRowContext(ctx, q, version)
	rev, err := scanRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, migrate.ErrRevisionNotExist
	}
	return rev, err
}

func (r *pgRevisionReadWriter) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	var partialJSON any
	if len(rev.PartialHashes) > 0 {
		b, err := json.Marshal(rev.PartialHashes)
		if err != nil {
			return err
		}
		partialJSON = string(b)
	}

	var errVal, errStmtVal any
	if rev.Error != "" {
		errVal = rev.Error
	}
	if rev.ErrorStmt != "" {
		errStmtVal = rev.ErrorStmt
	}

	hash := rev.Hash
	if hash != "" && !strings.HasPrefix(hash, "h1:") {
		hash = "h1:" + hash
	}

	const q = `
INSERT INTO "atlas_schema_revisions"."atlas_schema_revisions" (
  "version", "description", "type", "applied", "total", "executed_at", "execution_time",
  "error", "error_stmt", "hash", "partial_hashes", "operator_version"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT ("version") DO UPDATE SET
  "description" = EXCLUDED."description",
  "type" = EXCLUDED."type",
  "applied" = EXCLUDED."applied",
  "total" = EXCLUDED."total",
  "executed_at" = EXCLUDED."executed_at",
  "execution_time" = EXCLUDED."execution_time",
  "error" = EXCLUDED."error",
  "error_stmt" = EXCLUDED."error_stmt",
  "hash" = EXCLUDED."hash",
  "partial_hashes" = EXCLUDED."partial_hashes",
  "operator_version" = EXCLUDED."operator_version"`

	_, err := r.db.ExecContext(
		ctx,
		q,
		rev.Version,
		rev.Description,
		uint(rev.Type),
		rev.Applied,
		rev.Total,
		rev.ExecutedAt,
		int64(rev.ExecutionTime),
		errVal,
		errStmtVal,
		hash,
		partialJSON,
		rev.OperatorVersion,
	)
	return err
}

func (r *pgRevisionReadWriter) DeleteRevision(ctx context.Context, version string) error {
	const q = `DELETE FROM "atlas_schema_revisions"."atlas_schema_revisions" WHERE "version" = $1`
	_, err := r.db.ExecContext(ctx, q, version)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRevision(s rowScanner) (*migrate.Revision, error) {
	var (
		rev        migrate.Revision
		revType    uint
		execTimeNs int64
		partialRaw []byte
		storedHash string
	)
	err := s.Scan(
		&rev.Version,
		&rev.Description,
		&revType,
		&rev.Applied,
		&rev.Total,
		&rev.ExecutedAt,
		&execTimeNs,
		&rev.Error,
		&rev.ErrorStmt,
		&storedHash,
		&partialRaw,
		&rev.OperatorVersion,
	)
	if err != nil {
		return nil, err
	}
	rev.Type = migrate.RevisionType(revType)
	rev.ExecutionTime = time.Duration(execTimeNs)
	rev.Hash = strings.TrimPrefix(storedHash, "h1:")
	if len(partialRaw) > 0 {
		if err := json.Unmarshal(partialRaw, &rev.PartialHashes); err != nil {
			return nil, err
		}
	}
	return &rev, nil
}
