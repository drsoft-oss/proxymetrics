package duckdb

import (
	"path/filepath"
	"testing"
)

func TestSchema_MigratesAuditRunsColumnsOnOldDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.duckdb")

	// First open creates the table with the new columns via the bootstrap.
	s, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	// Drop the new columns to simulate a DB that pre-dates the migration.
	for _, col := range []string{"session_key", "unique_ip_count"} {
		if _, err := s.db.Exec(`ALTER TABLE audit_runs DROP COLUMN ` + col); err != nil {
			t.Fatalf("drop %s: %v", col, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Re-opening must run the ALTER ... ADD COLUMN IF NOT EXISTS migrations
	// and put both columns back.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	for _, col := range []string{"session_key", "unique_ip_count"} {
		var n int
		err := s2.db.QueryRow(
			`SELECT COUNT(*) FROM information_schema.columns
			 WHERE table_name = 'audit_runs' AND column_name = ?`, col,
		).Scan(&n)
		if err != nil {
			t.Fatalf("query %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("audit_runs.%s missing after migration", col)
		}
	}
}

func TestSchema_AuditTablesExist(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	for _, table := range []string{"audit_runs", "audit_requests", "geo_city_centroids"} {
		var n int
		err := s.db.QueryRow(
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, table,
		).Scan(&n)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if n != 1 {
			t.Fatalf("table %s missing", table)
		}
	}
}

func TestSchema_AuditRuns_HasSessionRotationColumns(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	for _, col := range []string{"session_key", "unique_ip_count"} {
		var n int
		err := s.db.QueryRow(
			`SELECT COUNT(*) FROM information_schema.columns
			 WHERE table_name = 'audit_runs' AND column_name = ?`, col,
		).Scan(&n)
		if err != nil {
			t.Fatalf("query %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("audit_runs.%s missing", col)
		}
	}
}
