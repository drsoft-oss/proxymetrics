package rollup

import (
	"context"
	"fmt"
	"time"
)

// RetentionPolicy is the configuration for a single Sweep.
type RetentionPolicy struct {
	EventsKeepFor   time.Duration
	Rollup1MinKeep  time.Duration
	Rollup1HourKeep time.Duration
	// 1day rollups kept forever.
}

// Sweeper is the slice of store.Store needed by Sweep.
type Sweeper interface {
	DeleteEventsBefore(ctx context.Context, ts time.Time) (int64, error)
	DeleteRollupsBefore(ctx context.Context, level string, ts time.Time) (int64, error)
}

// Sweep deletes rows older than the configured retention windows.
func Sweep(ctx context.Context, s Sweeper, p RetentionPolicy) error {
	now := time.Now().UTC()
	if _, err := s.DeleteEventsBefore(ctx, now.Add(-p.EventsKeepFor)); err != nil {
		return fmt.Errorf("sweep events: %w", err)
	}
	if _, err := s.DeleteRollupsBefore(ctx, "1min", now.Add(-p.Rollup1MinKeep)); err != nil {
		return fmt.Errorf("sweep 1min: %w", err)
	}
	if _, err := s.DeleteRollupsBefore(ctx, "1hour", now.Add(-p.Rollup1HourKeep)); err != nil {
		return fmt.Errorf("sweep 1hour: %w", err)
	}
	return nil
}
