// Package duckdb is the DuckDB-backed implementation of store.Store.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// Store is a DuckDB-backed store.Store. Methods are split across files in this package.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (creating if needed) the DuckDB file at path and runs the schema bootstrap.
func Open(path string) (*Store, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bootstrap schema: %w", err)
	}
	// Idempotent migrations for older DBs created before columns were dropped.
	for _, stmt := range migrations {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("migrate: %w (stmt: %s)", err, stmt)
		}
	}
	return &Store{db: db, path: path}, nil
}

// migrations is the list of idempotent ALTER statements applied after the
// CREATE TABLE bootstrap. Each statement must be safe to re-run on a fresh
// schema (use IF EXISTS guards).
var migrations = []string{
	`ALTER TABLE profiles DROP COLUMN IF EXISTS city`,
	`ALTER TABLE audit_runs ADD COLUMN IF NOT EXISTS session_key VARCHAR`,
	`ALTER TABLE audit_runs ADD COLUMN IF NOT EXISTS unique_ip_count INTEGER`,
	`ALTER TABLE audit_requests ADD COLUMN IF NOT EXISTS attempts INTEGER DEFAULT 1`,
}

func (s *Store) Close() error {
	return s.db.Close()
}

// QueryRollupSource exposes the underlying *sql.DB.QueryContext for the rollup
// package's compute SQL templates. Used by internal/rollup/compute.go.
func (s *Store) QueryRollupSource(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, q, args...)
}

// DB returns the underlying *sql.DB. Used by satellite packages (rollup,
// audit) that need raw queries beyond what the Store interface offers.
func (s *Store) DB() *sql.DB {
	return s.db
}
