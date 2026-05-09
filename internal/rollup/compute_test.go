package rollup_test

import (
	"context"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/rollup"
	"github.com/drsoft-oss/proxymetrics/internal/store"
	"github.com/drsoft-oss/proxymetrics/internal/store/sqlite"
)

func openTempStore(t *testing.T) *sqlite.Store {
	t.Helper()
	s, err := sqlite.Open(t.TempDir() + "/events.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func seedEvents(t *testing.T, s *sqlite.Store, events []store.Event) {
	t.Helper()
	if err := s.WriteEvents(context.Background(), events); err != nil {
		t.Fatal(err)
	}
}

func TestCompute1Min(t *testing.T) {
	s := openTempStore(t)
	ctx := context.Background()

	bucket := time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC)
	seedEvents(t, s, []store.Event{
		{TS: bucket.Add(10 * time.Second), RequestID: "r1", ProfileID: "p1", Vendor: "v", Type: "residential",
			TargetHost: "example.com", StatusCode: 200, StatusClass: "2xx",
			BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 0.001,
			Team: "growth", Project: "scrape"},
		{TS: bucket.Add(20 * time.Second), RequestID: "r2", ProfileID: "p1", Vendor: "v", Type: "residential",
			TargetHost: "example.com", StatusCode: 403, StatusClass: "4xx",
			BytesIn: 500, BytesOut: 50, LatencyMS: 30, CostUSD: 0.0005,
			Team: "growth", Project: "scrape"},
	})

	if err := rollup.Compute(ctx, s, "1min", bucket, bucket.Add(time.Minute)); err != nil {
		t.Fatalf("compute: %v", err)
	}

	rows, err := s.QueryRollups(ctx, store.RollupFilter{Level: "1min", From: bucket, To: bucket.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 { // one row per status_class
		t.Fatalf("rows: %d", len(rows))
	}
}

func TestCompute1Hour(t *testing.T) {
	s := openTempStore(t)
	ctx := context.Background()

	hour := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	events := []store.Event{}
	for i := 0; i < 60; i++ {
		events = append(events, store.Event{
			TS: hour.Add(time.Duration(i) * time.Minute), RequestID: "r",
			ProfileID: "p1", Vendor: "v", Type: "residential",
			StatusClass: "2xx", BytesIn: 100, BytesOut: 10, LatencyMS: 50, CostUSD: 0,
		})
	}
	seedEvents(t, s, events)

	if err := rollup.Compute(ctx, s, "1hour", hour, hour.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	rows, _ := s.QueryRollups(ctx, store.RollupFilter{Level: "1hour", From: hour, To: hour.Add(time.Hour)})
	if len(rows) != 1 || rows[0].RequestCount != 60 {
		t.Fatalf("got %+v", rows)
	}
}

func TestCompute_BadLevel(t *testing.T) {
	s := openTempStore(t)
	err := rollup.Compute(context.Background(), s, "bogus", time.Now(), time.Now().Add(time.Minute))
	if err == nil {
		t.Fatal("want error")
	}
}
