package audit_test

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/audit"
	"github.com/anonymous-proxies/proxymetrics/internal/geo"
	"github.com/anonymous-proxies/proxymetrics/internal/store/duckdb"
)

type stubResolver struct{ lat, lon float64 }

func (s stubResolver) Resolve(_ context.Context, _, _, _ string) (geo.Centroid, error) {
	return geo.Centroid{Lat: s.lat, Lon: s.lon}, nil
}

type stubLookup struct{ res geo.Result }

func (s stubLookup) Lookup(_ context.Context, _ *http.Client) (geo.Result, error) {
	return s.res, nil
}

func openTestManager(t *testing.T) (*audit.Manager, *audit.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	s, err := duckdb.Open(filepath.Join(dir, "t.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	as := audit.NewStore(s.DB())
	mgr := audit.NewManager(audit.ManagerConfig{
		Store:             as,
		Lookup:            stubLookup{res: geo.Result{IP: "1.1.1.1", Country: "RO", Source: "ipapi"}},
		Centroids:         stubResolver{lat: 44.43, lon: 26.10},
		MaxInFlight:       2,
		PerRequestTimeout: 2 * time.Second,
		MinRequestsPerRun: 1,
		MaxRequestsPerRun: 10,
	})
	return mgr, as, func() { _ = s.Close() }
}

func TestManager_Start_RejectsEmptyCountry(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	_, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL: "http://user:p@h:1", ExpectedType: "residential", RequestCount: 1,
	})
	if err == nil {
		t.Fatal("expected error: expected_country is required")
	}
}

func TestManager_Start_DerivesCheckLevelFromExpectedFields(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user:p@h:1",
		ExpectedCountry: "ro",
		ExpectedState:   "Bucharest",
		ExpectedType:    "residential",
		RequestCount:    1,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := as.GetRun(context.Background(), id)
		if got.Status == audit.StatusCompleted || got.Status == audit.StatusFailed {
			if got.CheckLevel != "state" {
				t.Fatalf("check_level: want state, got %q", got.CheckLevel)
			}
			if got.ExpectedCountry != "RO" {
				t.Fatalf("country: want RO (uppercased), got %q", got.ExpectedCountry)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("run did not finish in time")
}

func TestManager_Start_HappyPath(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
		ExpectedCountry: "RO",
		ExpectedState:   "Bucharest",
		ExpectedCity:    "Bucharest",
		ExpectedType:    "residential",
		RequestCount:    2,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := as.GetRun(context.Background(), id)
		if got.Status == audit.StatusCompleted {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("run did not complete in time")
}

func TestManager_Start_PrunesMapOnFinish(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
		ExpectedCountry: "RO",
		ExpectedState:   "Bucharest",
		ExpectedCity:    "Bucharest",
		ExpectedType:    "residential",
		RequestCount:    1,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := mgr.Get(id); !ok {
			return // pruned, success
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("runs map still contains the id after finish")
}

func TestManager_OrphanCleanup(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	now := time.Now().UTC()
	_ = as.InsertRun(context.Background(), audit.RunSummary{
		ID: "stale", StartedAt: now, Status: audit.StatusRunning,
		ProxyURL: "x", ExpectedLat: 0, ExpectedLon: 0,
		ExpectedType: "residential", CheckLevel: "country", RequestCount: 1,
	})
	if err := mgr.CleanupOrphans(context.Background()); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	got, _ := as.GetRun(context.Background(), "stale")
	if got.Status != audit.StatusFailed {
		t.Fatalf("status: %v", got.Status)
	}
}

func TestManager_Start_RejectsSessionKeyNotInURL(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	_, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-zone-residential:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
		SessionKey: "session",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error: %v", err)
	}
}

func TestManager_Start_RejectsSessionKeyAtLastToken(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	_, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-zone-session:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
		SessionKey: "session",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no following value") {
		t.Fatalf("error: %v", err)
	}
}

func TestManager_Start_AcceptsValidSessionKey(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-session-abc:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
		SessionKey: "session",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if id == "" {
		t.Fatal("empty id")
	}
}

func TestManager_Start_AbsentSessionKeyRunsAsToday(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-zone-residential:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
		// SessionKey unset
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if id == "" {
		t.Fatal("empty id")
	}
}

func TestManager_Start_PersistsSessionKey(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-session-abc:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
		SessionKey: "session",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	got, err := as.GetRun(context.Background(), id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SessionKey == nil || *got.SessionKey != "session" {
		t.Fatalf("session_key = %v", got.SessionKey)
	}
}

func TestManager_Start_Provider_SpecWins(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-provider-fromtag-country-RO:p@brd.superproxy.io:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
		Provider: "explicit",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	got, err := as.GetRun(context.Background(), id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Provider != "explicit" {
		t.Fatalf("provider = %q, want %q", got.Provider, "explicit")
	}
}

func TestManager_Start_Provider_CredtagWhenSpecEmpty(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-provider-fromtag-country-RO:p@brd.superproxy.io:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	got, _ := as.GetRun(context.Background(), id)
	if got.Provider != "fromtag" {
		t.Fatalf("provider = %q, want %q", got.Provider, "fromtag")
	}
}

func TestManager_Start_Provider_HostnameGuessWhenNoCredtag(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO:p@brd.superproxy.io:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	got, _ := as.GetRun(context.Background(), id)
	if got.Provider != "brightdata" {
		t.Fatalf("provider = %q, want %q", got.Provider, "brightdata")
	}
}

func TestManager_Start_Provider_EmptyWhenNothingResolves(t *testing.T) {
	mgr, as, cleanup := openTestManager(t)
	defer cleanup()
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO:p@unknown.example:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	got, _ := as.GetRun(context.Background(), id)
	if got.Provider != "" {
		t.Fatalf("provider = %q, want empty", got.Provider)
	}
}
