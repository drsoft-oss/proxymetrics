package audit_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"

	"github.com/anonymous-proxies/proxymetrics/internal/audit"
	"github.com/anonymous-proxies/proxymetrics/internal/store/duckdb"
)

func openTestStore(t *testing.T) (*audit.Store, *sql.DB, func()) {
	t.Helper()
	dir := t.TempDir()
	s, err := duckdb.Open(filepath.Join(dir, "test.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	as := audit.NewStore(s.DB())
	cleanup := func() { _ = s.Close() }
	return as, s.DB(), cleanup
}

func TestStore_InsertRun_SelectByID(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	sum := audit.RunSummary{
		ID: "01HXY1", StartedAt: now, Status: audit.StatusRunning,
		ProxyURL: "http://user-country-RO:***@h:1", ExpectedCountry: "RO",
		ExpectedLat: 44.43, ExpectedLon: 26.10, ExpectedType: "residential",
		CheckLevel: "country", RequestCount: 100,
	}
	if err := as.InsertRun(ctx, sum); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := as.GetRun(ctx, "01HXY1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != sum.ID || got.Status != sum.Status {
		t.Fatalf("got %+v", got)
	}
}

func TestStore_AppendRequestAndList(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "r1", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 1,
	}); err != nil {
		t.Fatalf("insert run: %v", err)
	}
	row := audit.RequestRow{
		RunID: "r1", Seq: 1, StartedAt: time.Now().UTC(), DurationMS: 123,
		ObservedIP: "1.1.1.1", ObservedCountry: "RO",
		LocationMatch: true, TypeMatch: true, GeoSource: "ipapi",
	}
	if err := as.AppendRequest(ctx, row); err != nil {
		t.Fatalf("append: %v", err)
	}
	got, err := as.ListRequests(ctx, "r1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Seq != 1 || got[0].ObservedIP != "1.1.1.1" {
		t.Fatalf("got %+v", got)
	}
}

func TestStore_FinaliseRun_UpdatesAggregates(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "r2", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 100,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	finished := time.Now().UTC()
	p50 := 100
	p95 := 200
	if err := as.FinaliseRun(ctx, audit.RunSummary{
		ID: "r2", FinishedAt: &finished, Status: audit.StatusCompleted,
		CompletedCount: 100, LocationMatchCount: 90, TypeMatchCount: 100, ErrorCount: 0,
		LatencyP50MS: &p50, LatencyP95MS: &p95, FallbackUsed: false,
	}); err != nil {
		t.Fatalf("finalise: %v", err)
	}
	got, err := as.GetRun(ctx, "r2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != audit.StatusCompleted || got.LocationMatchCount != 90 {
		t.Fatalf("got %+v", got)
	}
}

func TestStore_MarkOrphanedRunningFailed(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "r3", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 1,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := as.MarkOrphanedRunningFailed(ctx); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, err := as.GetRun(ctx, "r3")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != audit.StatusFailed {
		t.Fatalf("got %+v", got)
	}
}

func TestStore_CentroidCacheRoundtrip(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.PutCentroid(ctx, "RO", "Bucharest", "Bucharest", 44.43, 26.10); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, ok, err := as.GetCentroid(ctx, "RO", "Bucharest", "Bucharest")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.Lat != 44.43 || got.Lon != 26.10 {
		t.Fatalf("got %+v", got)
	}
	_, ok, _ = as.GetCentroid(ctx, "RO", "Bucharest", "NotCached")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestStore_ListRuns_OrderedByStartedAtDesc(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	earlier := time.Now().UTC().Add(-time.Hour)
	later := time.Now().UTC()
	for _, r := range []audit.RunSummary{
		{ID: "older", StartedAt: earlier, Status: audit.StatusRunning, ProxyURL: "x",
			ExpectedLat: 0, ExpectedLon: 0, ExpectedType: "residential", CheckLevel: "country", RequestCount: 100},
		{ID: "newer", StartedAt: later, Status: audit.StatusRunning, ProxyURL: "x",
			ExpectedLat: 0, ExpectedLon: 0, ExpectedType: "residential", CheckLevel: "country", RequestCount: 100},
	} {
		if err := as.InsertRun(ctx, r); err != nil {
			t.Fatalf("insert %s: %v", r.ID, err)
		}
	}

	rows, err := as.ListRuns(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 || rows[0].ID != "newer" || rows[1].ID != "older" {
		t.Fatalf("ordering wrong: %+v", rows)
	}

	// Pagination: limit=1 returns just the newest.
	page, err := as.ListRuns(ctx, 1, 0)
	if err != nil {
		t.Fatalf("page: %v", err)
	}
	if len(page) != 1 || page[0].ID != "newer" {
		t.Fatalf("page: %+v", page)
	}
}

func TestStore_PersistsSessionKey(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	sk := "sessionId"
	sum := audit.RunSummary{
		ID: "rsk", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedCountry: "RO", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 5,
		SessionKey: &sk,
	}
	if err := as.InsertRun(ctx, sum); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := as.GetRun(ctx, "rsk")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SessionKey == nil || *got.SessionKey != "sessionId" {
		t.Fatalf("session_key = %v", got.SessionKey)
	}
}

func TestStore_PersistsUniqueIPCount(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "ruic", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 3,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	finished := time.Now().UTC()
	count := 7
	if err := as.FinaliseRun(ctx, audit.RunSummary{
		ID: "ruic", FinishedAt: &finished, Status: audit.StatusCompleted,
		CompletedCount: 10, UniqueIPCount: &count,
	}); err != nil {
		t.Fatalf("finalise: %v", err)
	}
	got, err := as.GetRun(ctx, "ruic")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UniqueIPCount == nil || *got.UniqueIPCount != 7 {
		t.Fatalf("unique_ip_count = %v", got.UniqueIPCount)
	}
}

func TestStore_AppendRequest_RoundTripsAttempts(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "ra", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 1,
	}); err != nil {
		t.Fatalf("insert run: %v", err)
	}
	if err := as.AppendRequest(ctx, audit.RequestRow{
		RunID: "ra", Seq: 1, StartedAt: time.Now().UTC(), DurationMS: 12,
		Attempts: 2, GeoSource: "ipapi",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	rows, err := as.ListRequests(ctx, "ra")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].Attempts != 2 {
		t.Fatalf("expected attempts=2, got rows=%+v", rows)
	}
}

func TestStore_AppendRequest_ZeroAttemptsCoercesToOne(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "rz", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 1,
	}); err != nil {
		t.Fatalf("insert run: %v", err)
	}
	// Attempts is the Go zero value here; the store should coerce to 1
	// (atLeastOne) so the column never carries a misleading 0.
	if err := as.AppendRequest(ctx, audit.RequestRow{
		RunID: "rz", Seq: 1, StartedAt: time.Now().UTC(), DurationMS: 5,
		GeoSource: "ipapi",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	rows, err := as.ListRequests(ctx, "rz")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].Attempts != 1 {
		t.Fatalf("expected attempts=1, got rows=%+v", rows)
	}
}

func TestStore_OldRowsHaveNullSessionAndCount(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()

	// Insert without session key; fields default to nil/NULL.
	if err := as.InsertRun(ctx, audit.RunSummary{
		ID: "rold", StartedAt: time.Now().UTC(), Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 1,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := as.GetRun(ctx, "rold")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SessionKey != nil {
		t.Fatalf("session_key should be nil, got %v", *got.SessionKey)
	}
	if got.UniqueIPCount != nil {
		t.Fatalf("unique_ip_count should be nil, got %v", *got.UniqueIPCount)
	}
}
