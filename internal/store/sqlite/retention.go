package sqlite

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) DeleteEventsBefore(ctx context.Context, ts time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE ts < ?`, ts)
	if err != nil {
		return 0, fmt.Errorf("delete events: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *Store) DeleteRollupsBefore(ctx context.Context, level string, ts time.Time) (int64, error) {
	table := rollupTableForLevel(level)
	if table == "" {
		return 0, fmt.Errorf("rollups: invalid level %q", level)
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE ts_bucket < ?`, ts)
	if err != nil {
		return 0, fmt.Errorf("delete rollups: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
