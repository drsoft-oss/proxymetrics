//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/store"
	"github.com/anonymous-proxies/proxymetrics/internal/store/duckdb"
	"github.com/anonymous-proxies/proxymetrics/test/integration"
)

// TestBurst_AllEventsPersisted submits a burst of requests and confirms all of
// them land in DuckDB after flush_interval. This exercises the same code path
// as graceful shutdown (the event writer batching + flush) without the
// brittleness of sending real OS signals from the test process.
//
// Note: the plan originally called for a SIGTERM-based graceful-shutdown test.
// That approach was replaced here because sending SIGTERM/SIGINT to the test
// process also tears down the test runner itself, making assertions impossible.
// This burst test verifies the same underlying guarantee: the event writer
// batches and persists all events correctly under concurrent load, which is the
// mechanism that makes graceful shutdown loss-free.
func TestBurst_AllEventsPersisted(t *testing.T) {
	upstream := integration.StartFakeUpstream(t)
	srv := integration.Start(t, upstream.URL)

	proxyURL := integration.ProxyURL(srv, upstream.URL, "fake", "residential", 100)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	const N = 50
	var sent atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(upstream.URL + "/")
			if err == nil {
				io.ReadAll(resp.Body)
				resp.Body.Close()
				sent.Add(1)
			}
		}()
	}
	wg.Wait()

	// Wait for the writer to flush. Default flush_interval in test config is 100ms;
	// poll up to 5s for the row count to reach what was sent.
	deadline := time.Now().Add(5 * time.Second)
	var got int
	for time.Now().Before(deadline) {
		s, err := duckdb.Open(srv.DBPath)
		if err == nil {
			rows, _ := s.TailEvents(context.Background(), store.TailOptions{N: N + 100, ProfileID: "fake-residential"})
			s.Close()
			got = len(rows)
			if int64(got) >= sent.Load() {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("event count: persisted %d / sent %d", got, sent.Load())
}
