package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func (s *Store) DBStats(ctx context.Context) (store.DBStats, error) {
	out := store.DBStats{Rows: map[string]int64{}}

	tables := []string{"profiles", "events", "rollups_1min", "rollups_1hour", "rollups_1day"}
	for _, t := range tables {
		var n int64
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+t).Scan(&n); err != nil {
			return store.DBStats{}, fmt.Errorf("count %s: %w", t, err)
		}
		out.Rows[t] = n
	}

	var oldest, newest sql.NullTime
	if err := s.db.QueryRowContext(ctx, "SELECT MIN(ts), MAX(ts) FROM events").Scan(&oldest, &newest); err != nil {
		return store.DBStats{}, fmt.Errorf("event ts range: %w", err)
	}
	if oldest.Valid {
		out.OldestEventTS = oldest.Time
	}
	if newest.Valid {
		out.NewestEventTS = newest.Time
	}

	if s.path != "" {
		if fi, err := os.Stat(s.path); err == nil {
			out.SizeBytes = fi.Size()
		}
	}
	return out, nil
}

func (s *Store) Vacuum(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA force_checkpoint`); err != nil {
		return fmt.Errorf("force_checkpoint: %w", err)
	}
	return nil
}
