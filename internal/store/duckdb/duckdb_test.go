package duckdb_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
	"github.com/drsoft-oss/proxymetrics/internal/store/duckdb"
)

func TestOpen_CreatesSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.duckdb")

	s, err := duckdb.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	// Sanity: a no-op call should not error if the schema is in place.
	if _, err := s.ListProfiles(context.Background()); err != nil {
		t.Fatalf("ListProfiles after Open: %v", err)
	}
}

func TestOpen_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.duckdb")

	for i := 0; i < 2; i++ {
		s, err := duckdb.Open(path)
		if err != nil {
			t.Fatalf("Open #%d: %v", i, err)
		}
		s.Close()
	}
}

func openTemp(t *testing.T) *duckdb.Store {
	t.Helper()
	s, err := duckdb.Open(filepath.Join(t.TempDir(), "events.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func float64Ptr(v float64) *float64 { return &v }

func TestProfileCRUD(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	p := store.Profile{
		ID:                "prof_test",
		Label:             "Test",
		Vendor:            "custom",
		Type:              "residential",
		Region:            "us",
		UpstreamURL:       "http://user:pass@upstream.example:8000",
		PricePerGB:        float64Ptr(2.5),
		PricePerGBOverage: float64Ptr(5.0),
		IncludedGB:        float64Ptr(10),
		Currency:          "USD",
		DefaultTeam:       "growth",
		DefaultProject:    "amazon",
	}

	if err := s.CreateProfile(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.GetProfile(ctx, "prof_test")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Label != "Test" || got.Vendor != "custom" {
		t.Fatalf("get returned %+v", got)
	}
	if got.PricePerGB == nil || *got.PricePerGB != 2.5 {
		t.Fatalf("price_per_gb roundtrip failed: %+v", got.PricePerGB)
	}

	all, err := s.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("list len: got %d, want 1", len(all))
	}

	// PATCH — overwrite Label.
	p.Label = "Test 2"
	time.Sleep(2 * time.Millisecond) // ensure updated_at advances
	if err := s.UpdateProfile(ctx, p); err != nil {
		t.Fatalf("update: %v", err)
	}
	got2, _ := s.GetProfile(ctx, "prof_test")
	if got2.Label != "Test 2" {
		t.Fatalf("update did not persist: %q", got2.Label)
	}

	if err := s.DeleteProfile(ctx, "prof_test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = s.GetProfile(ctx, "prof_test")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("get after delete: want ErrNotFound, got %v", err)
	}
}

func TestCreateProfile_DuplicateID(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	p := store.Profile{ID: "dup", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "http://x", Currency: "USD"}
	if err := s.CreateProfile(ctx, p); err != nil {
		t.Fatal(err)
	}
	err := s.CreateProfile(ctx, p)
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("dup: want ErrConflict, got %v", err)
	}
}

func TestWriteEventsAndTail(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	// Profile required by FK convention (we don't enforce it in DuckDB,
	// but we still create it so future referential checks don't trip the test).
	_ = s.CreateProfile(ctx, store.Profile{
		ID: "p1", Label: "L", Vendor: "custom", Type: "residential",
		UpstreamURL: "http://u", Currency: "USD",
	})

	now := time.Now().UTC()
	batch := []store.Event{
		{TS: now, RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "example.com", StatusCode: 200, StatusClass: "2xx",
			BytesIn: 1024, BytesOut: 256, LatencyMS: 42, CostUSD: 0.001,
			Team: "growth", Project: "scrape"},
		{TS: now.Add(time.Second), RequestID: "r2", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "example.com", StatusCode: 403, StatusClass: "4xx",
			BytesIn: 512, BytesOut: 128, LatencyMS: 30, CostUSD: 0.0005,
			Team: "growth", Project: "scrape"},
	}
	if err := s.WriteEvents(ctx, batch); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := s.TailEvents(ctx, store.TailOptions{N: 10})
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("tail len: got %d, want 2", len(got))
	}
	// Newest first.
	if got[0].RequestID != "r2" || got[1].RequestID != "r1" {
		t.Fatalf("tail order wrong: %+v", got)
	}
}

func TestSumBytesInThisMonthByProfile(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "http://u", Currency: "USD"})

	now := time.Now().UTC()
	thisMonth := time.Date(now.Year(), now.Month(), 5, 0, 0, 0, 0, time.UTC)
	lastMonth := thisMonth.AddDate(0, -1, 0)

	if err := s.WriteEvents(ctx, []store.Event{
		{TS: thisMonth, RequestID: "a", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 1000, LatencyMS: 1, CostUSD: 0},
		{TS: thisMonth.Add(time.Hour), RequestID: "b", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 500, LatencyMS: 1, CostUSD: 0},
		{TS: lastMonth, RequestID: "c", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 9999, LatencyMS: 1, CostUSD: 0},
	}); err != nil {
		t.Fatal(err)
	}

	m, err := s.SumBytesInThisMonthByProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := m["p1"]; got != 1500 {
		t.Fatalf("month sum: got %d, want 1500", got)
	}
}

func TestOpen_RollupTablesPresent(t *testing.T) {
	s := openTemp(t)
	for _, level := range []string{"1min", "1hour", "1day"} {
		ts, err := s.LatestRollupBucket(context.Background(), level)
		if err != nil {
			t.Fatalf("LatestRollupBucket(%q): %v", level, err)
		}
		if !ts.IsZero() {
			t.Fatalf("expected zero time for empty %q, got %v", level, ts)
		}
	}
}

func TestRollups_WriteAndQuery(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	now := time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC)
	rows := []store.RollupRow{
		{TSBucket: now, ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "2xx", TargetHost: "example.com", Team: "growth", Project: "scrape",
			RequestCount: 10, BytesInTotal: 1000, BytesOutTotal: 200,
			LatencyMSAvg: 50, LatencyMSP50: 40, LatencyMSP95: 80, LatencyMSP99: 100,
			CostUSDTotal: 0.01, SuccessCount: 10, FailureCount: 0},
		{TSBucket: now, ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "4xx", TargetHost: "example.com", Team: "growth", Project: "scrape",
			RequestCount: 2, BytesInTotal: 100, BytesOutTotal: 30,
			LatencyMSAvg: 60, LatencyMSP50: 60, LatencyMSP95: 60, LatencyMSP99: 60,
			CostUSDTotal: 0.001, SuccessCount: 0, FailureCount: 2},
	}

	if err := s.WriteRollups(ctx, "1min", rows); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := s.QueryRollups(ctx, store.RollupFilter{Level: "1min", From: now, To: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("query len: got %d, want 2", len(got))
	}
}

func TestRollups_LatestBucket(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	got, err := s.LatestRollupBucket(ctx, "1min")
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsZero() {
		t.Fatalf("empty: got %v, want zero", got)
	}

	a := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	b := time.Date(2026, 4, 29, 12, 5, 0, 0, time.UTC)
	rows := []store.RollupRow{
		{TSBucket: a, ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx",
			RequestCount: 1, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0,
			SuccessCount: 1, FailureCount: 0},
		{TSBucket: b, ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx",
			RequestCount: 1, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0,
			SuccessCount: 1, FailureCount: 0},
	}
	if err := s.WriteRollups(ctx, "1min", rows); err != nil {
		t.Fatal(err)
	}

	got, err = s.LatestRollupBucket(ctx, "1min")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(b) {
		t.Fatalf("latest: got %v, want %v", got, b)
	}
}

func TestRollups_BadLevel(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_, err := s.LatestRollupBucket(ctx, "bogus")
	if err == nil {
		t.Fatal("want error for bogus level")
	}
}

func TestQueryEvents_FilterAndTotal(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	_ = s.CreateProfile(ctx, store.Profile{
		ID: "p1", Label: "L", Vendor: "custom", Type: "residential",
		UpstreamURL: "http://u", Currency: "USD",
	})

	now := time.Now().UTC()
	batch := make([]store.Event, 5)
	for i := range batch {
		batch[i] = store.Event{
			TS: now.Add(time.Duration(i) * time.Second), RequestID: fmt.Sprintf("r%d", i),
			ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "example.com", StatusCode: 200, StatusClass: "2xx",
			BytesIn: int64(1000 * (i + 1)), BytesOut: 100, LatencyMS: 50, CostUSD: 0,
			Team: "growth", Project: "scrape",
		}
	}
	if err := s.WriteEvents(ctx, batch); err != nil {
		t.Fatal(err)
	}

	rows, total, err := s.QueryEvents(ctx, store.EventFilter{
		ProfileIDs: []string{"p1"},
		Limit:      3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows: got %d, want 3", len(rows))
	}
	if total != 5 {
		t.Fatalf("total: got %d, want 5", total)
	}

	rows, total, err = s.QueryEvents(ctx, store.EventFilter{
		StatusClasses: []string{"4xx", "5xx"}, // none match
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 || total != 0 {
		t.Fatalf("filtered: got rows=%d total=%d, want 0/0", len(rows), total)
	}
}

func TestQueryEvents_DefaultLimit(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{
		ID: "p1", Label: "L", Vendor: "custom", Type: "residential",
		UpstreamURL: "http://u", Currency: "USD",
	})

	batch := make([]store.Event, 60)
	now := time.Now().UTC()
	for i := range batch {
		batch[i] = store.Event{
			TS: now.Add(time.Duration(i) * time.Second), RequestID: fmt.Sprintf("r%d", i),
			ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusCode: 200, StatusClass: "2xx",
			BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 0,
		}
	}
	_ = s.WriteEvents(ctx, batch)

	rows, total, err := s.QueryEvents(ctx, store.EventFilter{}) // no limit set
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 50 {
		t.Fatalf("default limit: got %d rows, want 50", len(rows))
	}
	if total != 60 {
		t.Fatalf("total: got %d, want 60", total)
	}
}

func TestDeleteEventsBefore(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})

	old := time.Now().UTC().AddDate(0, 0, -45)
	new := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: old, RequestID: "old", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
		{TS: new, RequestID: "new", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})

	deleted, err := s.DeleteEventsBefore(ctx, time.Now().UTC().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted: got %d, want 1", deleted)
	}

	rows, _, _ := s.QueryEvents(ctx, store.EventFilter{})
	if len(rows) != 1 || rows[0].RequestID != "new" {
		t.Fatalf("survivors: %+v", rows)
	}
}

func TestDeleteRollupsBefore(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	new := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = s.WriteRollups(ctx, "1min", []store.RollupRow{
		{TSBucket: old, ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx",
			RequestCount: 1, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0,
			SuccessCount: 1, FailureCount: 0},
		{TSBucket: new, ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx",
			RequestCount: 1, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0,
			SuccessCount: 1, FailureCount: 0},
	})

	cutoff := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	deleted, err := s.DeleteRollupsBefore(ctx, "1min", cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted: %d", deleted)
	}
}

func TestDeleteRollupsBefore_BadLevel(t *testing.T) {
	s := openTemp(t)
	_, err := s.DeleteRollupsBefore(context.Background(), "bogus", time.Now())
	if err == nil {
		t.Fatal("want error")
	}
}

func TestDBStats(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})

	stats, err := s.DBStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rows["events"] != 1 {
		t.Fatalf("events count: %d", stats.Rows["events"])
	}
	if stats.Rows["profiles"] != 1 {
		t.Fatalf("profiles count: %d", stats.Rows["profiles"])
	}
	if stats.OldestEventTS.IsZero() || stats.NewestEventTS.IsZero() {
		t.Fatalf("event timestamps: %+v", stats)
	}
}

func TestVacuum(t *testing.T) {
	s := openTemp(t)
	if err := s.Vacuum(context.Background()); err != nil {
		t.Fatalf("vacuum: %v", err)
	}
}

func TestDistinctDimensions_FixedStatusClasses(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	got, err := s.DistinctDimensions(ctx, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("DistinctDimensions: %v", err)
	}
	want := []string{"2xx", "3xx", "4xx", "5xx"}
	if len(got.StatusClasses) != 4 {
		t.Fatalf("StatusClasses: got %v, want %v", got.StatusClasses, want)
	}
	for i, sc := range want {
		if got.StatusClasses[i] != sc {
			t.Fatalf("StatusClasses[%d]: got %q, want %q", i, got.StatusClasses[i], sc)
		}
	}
	if len(got.Vendors) != 0 {
		t.Fatalf("empty store: Vendors should be empty, got %v", got.Vendors)
	}
}

func TestDistinctDimensions_FromEvents(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "Bright Data", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.CreateProfile(ctx, store.Profile{ID: "p2", Label: "Oxylabs", Vendor: "oxylabs", Type: "datacenter", UpstreamURL: "u", Currency: "USD"})

	now := time.Now().UTC()
	older := now.Add(-48 * time.Hour) // outside the [from,to) window

	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "r1", ProfileID: "p1", Vendor: "brightdata", Type: "residential", Region: "us-east", Team: "growth", Project: "amazon", StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
		{TS: now, RequestID: "r2", ProfileID: "p2", Vendor: "oxylabs", Type: "datacenter", Region: "eu-west", Team: "growth", Project: "ebay", StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
		{TS: older, RequestID: "old", ProfileID: "p1", Vendor: "smartproxy", Type: "mobile", Region: "asia", Team: "ops", Project: "old", StatusClass: "5xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})

	got, err := s.DistinctDimensions(ctx, now.Add(-24*time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Inside-window values only:
	if !equalSorted(got.Vendors, []string{"brightdata", "oxylabs"}) {
		t.Fatalf("Vendors: %v", got.Vendors)
	}
	if !equalSorted(got.Types, []string{"datacenter", "residential"}) {
		t.Fatalf("Types: %v", got.Types)
	}
	if !equalSorted(got.Regions, []string{"eu-west", "us-east"}) {
		t.Fatalf("Regions: %v", got.Regions)
	}
	if !equalSorted(got.Teams, []string{"growth"}) {
		t.Fatalf("Teams: %v", got.Teams)
	}
	if !equalSorted(got.Projects, []string{"amazon", "ebay"}) {
		t.Fatalf("Projects: %v", got.Projects)
	}

	// Profiles always returns the full registered set, regardless of window.
	gotIDs := make([]string, 0, len(got.Profiles))
	for _, p := range got.Profiles {
		gotIDs = append(gotIDs, p.ID)
	}
	if !equalSorted(gotIDs, []string{"p1", "p2"}) {
		t.Fatalf("Profiles ids: %v", gotIDs)
	}
}

func equalSorted(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestStatusCodeDistribution_OrderingAndAggregation(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "Bright Data", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.CreateProfile(ctx, store.Profile{ID: "p2", Label: "Smartproxy", Vendor: "smartproxy", Type: "datacenter", UpstreamURL: "u", Currency: "USD"})

	now := time.Now().UTC()
	events := []store.Event{
		// 200 dominated by brightdata (3 vs 1)
		{TS: now, RequestID: "a", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: now, RequestID: "b", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: now, RequestID: "c", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: now, RequestID: "d", ProfileID: "p2", Vendor: "smartproxy", Type: "datacenter", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		// 429 dominated by smartproxy (2 vs 0)
		{TS: now, RequestID: "e", ProfileID: "p2", Vendor: "smartproxy", Type: "datacenter", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
		{TS: now, RequestID: "f", ProfileID: "p2", Vendor: "smartproxy", Type: "datacenter", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
	}
	if err := s.WriteEvents(ctx, events); err != nil {
		t.Fatal(err)
	}

	rows, err := s.StatusCodeDistribution(ctx, store.EventFilter{From: now.Add(-time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("StatusCodeDistribution: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows: got %d, want 2 (200, 429)", len(rows))
	}

	// Order: descending by Requests; 200 has 4, 429 has 2.
	if rows[0].Code != 200 || rows[0].Requests != 4 {
		t.Fatalf("rows[0]: %+v", rows[0])
	}
	if rows[1].Code != 429 || rows[1].Requests != 2 {
		t.Fatalf("rows[1]: %+v", rows[1])
	}

	// Wasted only counted on non-2xx.
	if rows[0].WastedUSD != 0 {
		t.Fatalf("rows[0].WastedUSD: got %v, want 0", rows[0].WastedUSD)
	}
	if rows[0].SpendUSD != 4.0 {
		t.Fatalf("rows[0].SpendUSD: got %v, want 4.0", rows[0].SpendUSD)
	}
	if rows[1].WastedUSD != 1.0 {
		t.Fatalf("rows[1].WastedUSD: got %v, want 1.0", rows[1].WastedUSD)
	}

	// TopProviderVendor.
	if rows[0].TopProviderVendor != "brightdata" {
		t.Fatalf("rows[0].TopProviderVendor: %q", rows[0].TopProviderVendor)
	}
	if rows[1].TopProviderVendor != "smartproxy" {
		t.Fatalf("rows[1].TopProviderVendor: %q", rows[1].TopProviderVendor)
	}

	// Class derived from event row.
	if rows[0].Class != "2xx" || rows[1].Class != "4xx" {
		t.Fatalf("classes: %+v / %+v", rows[0], rows[1])
	}
}

func TestStatusCodeDetail_NotFound(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_, err := s.StatusCodeDetail(ctx, 429, store.EventFilter{
		From: time.Now().Add(-time.Hour),
		To:   time.Now().Add(time.Hour),
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("empty store: want ErrNotFound, got %v", err)
	}
}

func TestStatusCodeDetail_AggregatesAndTopLists(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L1", Vendor: "smartproxy", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.CreateProfile(ctx, store.Profile{ID: "p2", Label: "L2", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})

	now := time.Now().UTC()
	// 4 x 429 from smartproxy (3 to api.example.com, 1 to other.example.com)
	// 1 x 429 from brightdata (to api.example.com)
	// 1 x 200 noise (must be excluded)
	events := []store.Event{
		{TS: now, RequestID: "a", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", TargetHost: "api.example.com", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.50},
		{TS: now, RequestID: "b", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", TargetHost: "api.example.com", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.50},
		{TS: now, RequestID: "c", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", TargetHost: "api.example.com", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.50},
		{TS: now, RequestID: "d", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", TargetHost: "other.example.com", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.50},
		{TS: now, RequestID: "e", ProfileID: "p2", Vendor: "brightdata", Type: "residential", TargetHost: "api.example.com", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.20},
		{TS: now, RequestID: "f", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", TargetHost: "api.example.com", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.00},
	}
	if err := s.WriteEvents(ctx, events); err != nil {
		t.Fatal(err)
	}

	got, err := s.StatusCodeDetail(ctx, 429, store.EventFilter{
		From: now.Add(-time.Hour), To: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("StatusCodeDetail: %v", err)
	}
	if got.Code != 429 || got.Class != "4xx" {
		t.Fatalf("code/class: %+v", got)
	}
	if got.Requests != 5 {
		t.Fatalf("requests: %d, want 5", got.Requests)
	}
	wantWasted := 4*0.50 + 0.20
	if abs(got.WastedUSD-wantWasted) > 1e-9 {
		t.Fatalf("wasted: %v, want %v", got.WastedUSD, wantWasted)
	}
	if abs(got.SpendUSD-wantWasted) > 1e-9 {
		t.Fatalf("spend (==wasted for 4xx): %v, want %v", got.SpendUSD, wantWasted)
	}

	// Top providers: smartproxy=4 first, brightdata=1 second.
	if len(got.TopProviders) != 2 {
		t.Fatalf("TopProviders len: %d", len(got.TopProviders))
	}
	if got.TopProviders[0].Vendor != "smartproxy" || got.TopProviders[0].Requests != 4 {
		t.Fatalf("TopProviders[0]: %+v", got.TopProviders[0])
	}
	if got.TopProviders[1].Vendor != "brightdata" || got.TopProviders[1].Requests != 1 {
		t.Fatalf("TopProviders[1]: %+v", got.TopProviders[1])
	}

	// Top targets: api.example.com=4, other.example.com=1.
	if len(got.TopTargets) != 2 {
		t.Fatalf("TopTargets len: %d", len(got.TopTargets))
	}
	if got.TopTargets[0].Host != "api.example.com" || got.TopTargets[0].Requests != 4 {
		t.Fatalf("TopTargets[0]: %+v", got.TopTargets[0])
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestRequestsByHour_GroupByProfile(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "v", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.CreateProfile(ctx, store.Profile{ID: "p2", Label: "L", Vendor: "v", Type: "residential", UpstreamURL: "u", Currency: "USD"})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) // pinned for determinism
	// Use rollups_1hour because RequestsByHour reads from there.
	rows := []store.RollupRow{
		{TSBucket: now.Add(-2 * time.Hour), ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx", RequestCount: 10, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 10, FailureCount: 0},
		{TSBucket: now.Add(-2 * time.Hour), ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "4xx", RequestCount: 2, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 0, FailureCount: 2},
		{TSBucket: now.Add(-1 * time.Hour), ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx", RequestCount: 7, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 7, FailureCount: 0},
		{TSBucket: now.Add(-1 * time.Hour), ProfileID: "p2", Vendor: "v", Type: "residential", StatusClass: "2xx", RequestCount: 5, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 5, FailureCount: 0},
		// Outside trailing 24h — must be excluded.
		{TSBucket: now.Add(-30 * time.Hour), ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx", RequestCount: 999, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 999, FailureCount: 0},
	}
	if err := s.WriteRollups(ctx, "1hour", rows); err != nil {
		t.Fatal(err)
	}

	got, err := s.RequestsByHour(ctx, "profile_id", now)
	if err != nil {
		t.Fatalf("RequestsByHour: %v", err)
	}

	// Build a (key, hour-offset) → count map for assertions, ignoring zero rows.
	type key struct {
		k    string
		offH int
	}
	m := map[key]int64{}
	for _, b := range got {
		offH := int(now.Sub(b.Hour) / time.Hour)
		m[key{b.Key, offH}] = b.Count
	}

	if m[key{"p1", 2}] != 12 {
		t.Fatalf("p1 -2h: got %d, want 12 (10+2)", m[key{"p1", 2}])
	}
	if m[key{"p1", 1}] != 7 {
		t.Fatalf("p1 -1h: got %d, want 7", m[key{"p1", 1}])
	}
	if m[key{"p2", 1}] != 5 {
		t.Fatalf("p2 -1h: got %d, want 5", m[key{"p2", 1}])
	}
	if _, present := m[key{"p1", 30}]; present {
		t.Fatalf("excluded row leaked: %+v", m)
	}
}

func TestRequestsByHour_BadGroupBy(t *testing.T) {
	s := openTemp(t)
	_, err := s.RequestsByHour(context.Background(), "bogus", time.Now())
	if err == nil {
		t.Fatal("want error for bogus groupBy")
	}
}

func TestStatusCodeDistribution_HonorsFilter(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})

	now := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "a", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: now, RequestID: "b", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusCode: 500, StatusClass: "5xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
	})

	rows, err := s.StatusCodeDistribution(ctx, store.EventFilter{
		From:          now.Add(-time.Hour),
		To:            now.Add(time.Hour),
		StatusClasses: []string{"5xx"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Code != 500 {
		t.Fatalf("filter: got %+v", rows)
	}
}
