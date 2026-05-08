package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elazarl/goproxy"

	"github.com/drsoft-oss/proxymetrics/internal/audit"
	"github.com/drsoft-oss/proxymetrics/internal/geo"
	"github.com/drsoft-oss/proxymetrics/internal/store/duckdb"
)

func TestIntegration_GeoAudit_EndToEnd(t *testing.T) {
	// Fake ipapi.is upstream
	ipapi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ip":"203.0.113.7",
			"is_datacenter":false,"is_mobile":false,
			"location":{"country_code":"RO","state":"Bucharest","city":"Bucharest","latitude":44.43,"longitude":26.10}
		}`))
	}))
	defer ipapi.Close()

	// Tiny proxy
	prox := goproxy.NewProxyHttpServer()
	proxSrv := httptest.NewServer(prox)
	defer proxSrv.Close()

	// Audit manager wired to the fake upstream and a fake centroid resolver.
	dir := t.TempDir()
	s, err := duckdb.Open(filepath.Join(dir, "i.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer s.Close()

	mgr := audit.NewManager(audit.ManagerConfig{
		Store:             audit.NewStore(s.DB()),
		Lookup:            geo.NewIPAPIIs(ipapi.URL),
		Centroids:         stubResolver{lat: 44.43, lon: 26.10},
		MaxInFlight:       3,
		PerRequestTimeout: 5 * time.Second,
		MinRequestsPerRun: 1,
		MaxRequestsPerRun: 100,
	})

	// Use the test goproxy as the credtag URL (no actual creds enforced).
	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@" + proxSrv.Listener.Addr().String(),
		ExpectedCountry: "RO",
		ExpectedState:   "Bucharest",
		ExpectedCity:    "Bucharest",
		ExpectedType:    "residential",
		RequestCount:    5,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	// Poll for completion via the manager's HTTP endpoints (mounted on a mux).
	mux := http.NewServeMux()
	audit.RegisterRoutes(mux, mgr)
	api := httptest.NewServer(mux)
	defer api.Close()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, _ := http.Get(api.URL + "/api/v1/audits/" + id)
		if resp.StatusCode == http.StatusOK {
			var d audit.RunDetail
			_ = json.NewDecoder(resp.Body).Decode(&d)
			if d.Run.Status == audit.StatusCompleted {
				if d.Run.LocationMatchCount != 5 {
					t.Fatalf("matches: %+v", d.Run)
				}
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("audit did not complete in time")
}

func TestIntegration_GeoAudit_RotatesSessionPerRequest(t *testing.T) {
	ipapi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ip":"203.0.113.7",
			"is_datacenter":false,"is_mobile":false,
			"location":{"country_code":"RO","state":"Bucharest","city":"Bucharest","latitude":44.43,"longitude":26.10}
		}`))
	}))
	defer ipapi.Close()

	// Capture the auth username on every CONNECT/GET that hits the proxy.
	prox := goproxy.NewProxyHttpServer()
	var (
		mu        sync.Mutex
		usernames []string
	)
	prox.OnRequest().DoFunc(func(r *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if h := r.Header.Get("Proxy-Authorization"); h != "" {
			mu.Lock()
			usernames = append(usernames, h)
			mu.Unlock()
		}
		return r, nil
	})
	proxSrv := httptest.NewServer(prox)
	defer proxSrv.Close()

	dir := t.TempDir()
	s, err := duckdb.Open(filepath.Join(dir, "i.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	mgr := audit.NewManager(audit.ManagerConfig{
		Store:             audit.NewStore(s.DB()),
		Lookup:            geo.NewIPAPIIs(ipapi.URL),
		Centroids:         stubResolver{lat: 44.43, lon: 26.10},
		MaxInFlight:       3,
		PerRequestTimeout: 5 * time.Second,
		MinRequestsPerRun: 1,
		MaxRequestsPerRun: 100,
	})

	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL: "http://user-zone-residential-session-original:p@" +
			proxSrv.Listener.Addr().String(),
		ExpectedCountry: "RO",
		ExpectedState:   "Bucharest",
		ExpectedCity:    "Bucharest",
		ExpectedType:    "residential",
		RequestCount:    5,
		SessionKey:      "session",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	mux := http.NewServeMux()
	audit.RegisterRoutes(mux, mgr)
	api := httptest.NewServer(mux)
	defer api.Close()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, _ := http.Get(api.URL + "/api/v1/audits/" + id)
		if resp.StatusCode == http.StatusOK {
			var d audit.RunDetail
			_ = json.NewDecoder(resp.Body).Decode(&d)
			if d.Run.Status == audit.StatusCompleted {
				mu.Lock()
				captured := append([]string(nil), usernames...)
				mu.Unlock()
				if len(captured) < 5 {
					t.Fatalf("captured %d auth headers, expected 5", len(captured))
				}
				seen := map[string]struct{}{}
				for _, h := range captured {
					if strings.Contains(h, "session-original") {
						t.Fatalf("auth still has 'session-original': %q", h)
					}
					seen[h] = struct{}{}
				}
				if len(seen) != len(captured) {
					t.Fatalf("expected unique auth headers, got %d unique of %d total",
						len(seen), len(captured))
				}
				return
			}
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("audit did not complete in time")
}

type stubResolver struct{ lat, lon float64 }

func (s stubResolver) Resolve(_ context.Context, _, _, _ string) (geo.Centroid, error) {
	return geo.Centroid{Lat: s.lat, Lon: s.lon}, nil
}
