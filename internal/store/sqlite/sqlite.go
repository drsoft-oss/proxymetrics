// Package sqlite is the SQLite-backed implementation of store.Store.
//
// Driver: modernc.org/sqlite (pure Go, no CGO).
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed store.Store. Methods are split across files in this package.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (creating if needed) the SQLite file at path and runs the schema bootstrap.
//
// If a sibling events.duckdb file is present, Open refuses to start: this product
// shipped a DuckDB-backed store before the v0.x cutover and the on-disk format is
// incompatible. Operators must remove or archive the old file explicitly.
func Open(path string) (*Store, error) {
	if err := refuseLegacyDuckDB(path); err != nil {
		return nil, err
	}

	// _pragma options are applied on every connection in the pool by modernc.
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bootstrap schema: %w", err)
	}
	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, path: path}, nil
}

func refuseLegacyDuckDB(path string) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	// Only check when the new path is the canonical events.db; otherwise tests
	// using arbitrary tempdir filenames would all trip this guard.
	if base != "events.db" {
		return nil
	}
	legacy := filepath.Join(dir, "events.duckdb")
	if _, err := os.Stat(legacy); err == nil {
		return fmt.Errorf("found legacy %s alongside new %s. proxymetrics switched from DuckDB to SQLite; the on-disk format is incompatible. Move or delete events.duckdb to continue (no in-place upgrade is provided)", legacy, path)
	}
	return nil
}

// runMigrations applies idempotent ALTER statements after the CREATE TABLE
// bootstrap, by inspecting pragma_table_info first (SQLite has no
// "ADD COLUMN IF NOT EXISTS").
func runMigrations(db *sql.DB) error {
	type addCol struct {
		table  string
		column string
		ddl    string
	}
	adds := []addCol{
		{"audit_runs", "session_key", `ALTER TABLE audit_runs ADD COLUMN session_key TEXT`},
		{"audit_runs", "unique_ip_count", `ALTER TABLE audit_runs ADD COLUMN unique_ip_count INTEGER`},
		{"audit_requests", "attempts", `ALTER TABLE audit_requests ADD COLUMN attempts INTEGER DEFAULT 1`},
	}
	for _, a := range adds {
		has, err := tableHasColumn(db, a.table, a.column)
		if err != nil {
			return fmt.Errorf("migrate check %s.%s: %w", a.table, a.column, err)
		}
		if has {
			continue
		}
		if _, err := db.Exec(a.ddl); err != nil {
			return fmt.Errorf("migrate add %s.%s: %w", a.table, a.column, err)
		}
	}

	// DROP COLUMN IF EXISTS — SQLite supports DROP COLUMN since 3.35 but no IF EXISTS.
	type dropCol struct{ table, column string }
	drops := []dropCol{
		{"profiles", "city"},
	}
	for _, d := range drops {
		has, err := tableHasColumn(db, d.table, d.column)
		if err != nil {
			return fmt.Errorf("migrate check drop %s.%s: %w", d.table, d.column, err)
		}
		if !has {
			continue
		}
		if _, err := db.Exec(`ALTER TABLE ` + d.table + ` DROP COLUMN ` + d.column); err != nil {
			return fmt.Errorf("migrate drop %s.%s: %w", d.table, d.column, err)
		}
	}
	return nil
}

func tableHasColumn(db *sql.DB, table, column string) (bool, error) {
	var n int
	// pragma_table_info is a table-valued function in SQLite; safe against arbitrary
	// table identifiers because pragma_table_info(?) takes the table name as a value.
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
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

// isUniqueViolation reports whether err is a SQLite UNIQUE/PK constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// nullTimeText scans a SQLite TEXT/DATETIME value (typically from an aggregate
// like MAX(ts) where the driver loses column-type info and returns string)
// into a *time.Time. Use as the Scan destination, then read .Time / .Valid.
type nullTimeText struct {
	Time  time.Time
	Valid bool
}

func (n *nullTimeText) Scan(v any) error {
	if v == nil {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	switch x := v.(type) {
	case time.Time:
		n.Time, n.Valid = x, true
		return nil
	case string:
		return n.parseString(x)
	case []byte:
		return n.parseString(string(x))
	default:
		return fmt.Errorf("nullTimeText: unsupported scan source %T", v)
	}
}

func (n *nullTimeText) parseString(s string) error {
	if s == "" {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	// modernc.org/sqlite stores time.Time via fmt.Sprintf("%s", t) which uses
	// Go's default format ("2006-01-02 15:04:05.999999999 -0700 MST"). For
	// columns declared DATETIME the driver round-trips back to time.Time, but
	// aggregates (MAX, MIN) lose column-type info and surface as raw TEXT —
	// hence the need for this fallback parser.
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			n.Time, n.Valid = t.UTC(), true
			return nil
		}
	}
	return fmt.Errorf("nullTimeText: cannot parse %q", s)
}
