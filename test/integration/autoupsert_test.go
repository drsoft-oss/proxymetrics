//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store/duckdb"
	"github.com/anonymous-proxies/proxymetrics/test/integration"
)

// TestAutoUpsertProfile_FromObservedTraffic asserts that when N requests with
// a previously-unseen (provider, type) combo land, the registry persists
// exactly one profile row for that combo. No admin call required.
func TestAutoUpsertProfile_FromObservedTraffic(t *testing.T) {
	upstream := integration.StartFakeUpstream(t)
	srv := integration.Start(t, upstream.URL)

	proxyURL := integration.ProxyURL(srv, upstream.URL, "novelvendor", "datacenter", 250)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	for range 5 {
		resp, err := client.Get(upstream.URL + "/")
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		resp.Body.Close()
	}

	// Wait for at least one event to be persisted (writer flush + observer call).
	_ = integration.EventsAfter(t, srv, 5*time.Second)

	wantID := "novelvendor-datacenter"
	deadline := time.Now().Add(3 * time.Second)
	for {
		s, err := duckdb.Open(srv.DBPath)
		if err == nil {
			profiles, _ := s.ListProfiles(context.Background())
			s.Close()
			matches := 0
			for _, p := range profiles {
				if p.ID == wantID {
					matches++
					if p.Vendor != "novelvendor" {
						t.Errorf("vendor: got %q want %q", p.Vendor, "novelvendor")
					}
					if p.Type != "datacenter" {
						t.Errorf("type: got %q want %q", p.Type, "datacenter")
					}
				}
			}
			if matches == 1 {
				return
			}
			if matches > 1 {
				t.Fatalf("auto-upsert created %d rows for %q; want exactly 1", matches, wantID)
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("auto-upsert did not produce a row for %q within deadline", wantID)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
