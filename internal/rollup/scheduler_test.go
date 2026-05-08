package rollup_test

import (
	"context"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/rollup"
	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

type silentLogger struct{}

func (silentLogger) Debug(string, ...any) {}
func (silentLogger) Info(string, ...any)  {}
func (silentLogger) Warn(string, ...any)  {}
func (silentLogger) Error(string, ...any) {}

func TestScheduler_RunsTickAndExitsOnContextCancel(t *testing.T) {
	s := openTempStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Seed within the scheduler's first-run backfill window (24h for 1min).
	bucket := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Minute)
	seedEvents(t, s, []store.Event{
		{TS: bucket.Add(10 * time.Second), RequestID: "r1", ProfileID: "p1",
			Vendor: "v", Type: "residential", StatusClass: "2xx",
			BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})

	sched := rollup.NewScheduler(s, rollup.SchedulerConfig{
		TickEvery:      50 * time.Millisecond,
		Logger:         silentLogger{},
		ForceAllLevels: true,
	})

	done := make(chan error, 1)
	go func() { done <- sched.Run(ctx) }()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	rows, _ := s.QueryRollups(context.Background(), store.RollupFilter{Level: "1min"})
	if len(rows) == 0 {
		t.Fatal("expected at least one rollup_1min row after a tick")
	}
}
