package rollup_test

import (
	"context"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/rollup"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func TestSweep_RemovesOldEvents(t *testing.T) {
	s := openTempStore(t)
	ctx := context.Background()

	old := time.Now().UTC().AddDate(0, 0, -45)
	new := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: old, RequestID: "old", ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx",
			BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
		{TS: new, RequestID: "new", ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx",
			BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})

	if err := rollup.Sweep(ctx, s, rollup.RetentionPolicy{
		EventsKeepFor:   30 * 24 * time.Hour,
		Rollup1MinKeep:  90 * 24 * time.Hour,
		Rollup1HourKeep: 365 * 24 * time.Hour,
	}); err != nil {
		t.Fatal(err)
	}

	rows, _, _ := s.QueryEvents(ctx, store.EventFilter{})
	if len(rows) != 1 || rows[0].RequestID != "new" {
		t.Fatalf("survivors: %+v", rows)
	}
}
