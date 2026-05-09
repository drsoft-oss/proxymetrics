//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
	"github.com/drsoft-oss/proxymetrics/internal/store/sqlite"
	"github.com/drsoft-oss/proxymetrics/test/integration"
)

func TestE2E_RawEventsThenOverview(t *testing.T) {
	upstream := integration.StartFakeUpstream(t)
	srv := integration.Start(t, upstream.URL)

	proxyURL := integration.ProxyURL(srv, upstream.URL, "fake", "residential", 100)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	for i := 0; i < 5; i++ {
		resp, err := client.Get(upstream.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	deadline := time.Now().Add(5 * time.Second)
	var rows []store.Event
	for time.Now().Before(deadline) {
		s, err := sqlite.Open(srv.DBPath)
		if err == nil {
			rows, _ = s.TailEvents(context.Background(), store.TailOptions{N: 50})
			s.Close()
		}
		if len(rows) >= 5 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(rows) < 5 {
		t.Fatalf("rows: %d", len(rows))
	}

	resp, err := http.Get("http://" + srv.APIAddr + "/api/v1/overview")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body struct {
		RequestCount int64   `json:"request_count"`
		SuccessRate  float64 `json:"success_rate"`
		Source       string  `json:"source"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.RequestCount < 5 {
		t.Fatalf("overview request_count: %d", body.RequestCount)
	}
	if body.Source != "events" {
		t.Fatalf("source: %q", body.Source)
	}
}
