package rollup

import (
	"context"
	"sync/atomic"
	"time"
)

// Logger is the minimal logging surface the scheduler uses.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SchedulerStore is the slice of store.Store needed by the scheduler.
type SchedulerStore interface {
	Computer
	DBExec
	Sweeper
	LatestRollupBucket(ctx context.Context, level string) (time.Time, error)
}

// SchedulerConfig configures the rollup scheduler goroutine.
type SchedulerConfig struct {
	// TickEvery is the polling interval. Production uses 60s; tests can shrink it.
	TickEvery time.Duration

	// Retention for the hourly sweep.
	Retention RetentionPolicy

	// Logger receives rollup outcomes; nil → discard.
	Logger Logger

	// ForceAllLevels makes every tick run 1min, 1hour, 1day, AND sweep.
	// Used by tests so they don't have to wait for wall-clock alignment.
	ForceAllLevels bool
}

// Scheduler is a goroutine handle.
type Scheduler struct {
	s   SchedulerStore
	cfg SchedulerConfig

	running atomic.Bool
}

func NewScheduler(s SchedulerStore, cfg SchedulerConfig) *Scheduler {
	if cfg.TickEvery <= 0 {
		cfg.TickEvery = time.Minute
	}
	if cfg.Retention.EventsKeepFor <= 0 {
		cfg.Retention.EventsKeepFor = 30 * 24 * time.Hour
	}
	if cfg.Retention.Rollup1MinKeep <= 0 {
		cfg.Retention.Rollup1MinKeep = 90 * 24 * time.Hour
	}
	if cfg.Retention.Rollup1HourKeep <= 0 {
		cfg.Retention.Rollup1HourKeep = 365 * 24 * time.Hour
	}
	if cfg.Logger == nil {
		cfg.Logger = silentLog{}
	}
	return &Scheduler{s: s, cfg: cfg}
}

// Run blocks until ctx is canceled.
func (sc *Scheduler) Run(ctx context.Context) error {
	tick := time.NewTicker(sc.cfg.TickEvery)
	defer tick.Stop()

	// Best-effort initial run on startup.
	sc.tickOnce(ctx, time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-tick.C:
			sc.tickOnce(ctx, now.UTC())
		}
	}
}

// tickOnce performs one cycle. Re-entrant: if the previous tick is still running,
// this one is skipped and a WARN is logged.
func (sc *Scheduler) tickOnce(ctx context.Context, now time.Time) {
	if !sc.running.CompareAndSwap(false, true) {
		sc.cfg.Logger.Warn("rollup tick skipped; previous tick still running")
		return
	}
	defer sc.running.Store(false)

	sc.runLevel(ctx, "1min", now)

	if sc.cfg.ForceAllLevels || now.Minute()%5 == 0 {
		sc.runLevel(ctx, "1hour", now)
	}
	if sc.cfg.ForceAllLevels || now.Minute() == 0 {
		sc.runLevel(ctx, "1day", now)
		if err := Sweep(ctx, sc.s, sc.cfg.Retention); err != nil {
			sc.cfg.Logger.Warn("retention sweep failed", "err", err.Error())
		}
	}
}

func (sc *Scheduler) runLevel(ctx context.Context, level string, now time.Time) {
	latest, err := sc.s.LatestRollupBucket(ctx, level)
	if err != nil {
		sc.cfg.Logger.Warn("rollup latest bucket lookup failed", "level", level, "err", err.Error())
		return
	}

	to := nowBucket(level, now)
	var from time.Time
	if latest.IsZero() {
		// First run: pick a bounded backfill window so we don't scan all-time.
		switch level {
		case "1min":
			from = to.Add(-24 * time.Hour)
		case "1hour":
			from = to.Add(-7 * 24 * time.Hour)
		case "1day":
			from = to.AddDate(0, 0, -30)
		}
	} else {
		from = nextBucket(level, latest)
	}

	if !from.Before(to) {
		return
	}

	if err := Compute(ctx, sc.s, level, from, to); err != nil {
		sc.cfg.Logger.Warn("rollup compute failed", "level", level, "from", from, "to", to, "err", err.Error())
		return
	}
	sc.cfg.Logger.Debug("rollup computed", "level", level, "from", from, "to", to)
}

// nextBucket returns the start of the bucket immediately after `latest`.
func nextBucket(level string, latest time.Time) time.Time {
	switch level {
	case "1min":
		return latest.Add(time.Minute)
	case "1hour":
		return latest.Add(time.Hour)
	case "1day":
		return latest.AddDate(0, 0, 1)
	}
	return time.Time{}
}

// nowBucket returns the start of the bucket containing `now`. It is used as an
// EXCLUSIVE upper bound for the rollup window, so callers compute rollups for
// every bucket strictly before this — i.e., every bucket already completed.
func nowBucket(level string, now time.Time) time.Time {
	switch level {
	case "1min":
		return now.Truncate(time.Minute)
	case "1hour":
		return now.Truncate(time.Hour)
	case "1day":
		return now.Truncate(24 * time.Hour)
	}
	return time.Time{}
}

type silentLog struct{}

func (silentLog) Debug(string, ...any) {}
func (silentLog) Info(string, ...any)  {}
func (silentLog) Warn(string, ...any)  {}
func (silentLog) Error(string, ...any) {}
