package audit_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/audit"
)

func TestSSE_StreamsThenFinishes(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()

	mux := http.NewServeMux()
	audit.RegisterRoutes(mux, mgr)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:   "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 2,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	resp, err := http.Get(srv.URL + "/api/v1/audits/" + id + "/events")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	br := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(3 * time.Second)
	requestSeen := false
	finishedSeen := false
	for time.Now().Before(deadline) && !finishedSeen {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(strings.TrimSpace(line), "data: ")
			if strings.Contains(payload, `"type":"request"`) {
				requestSeen = true
			}
			if strings.Contains(payload, `"type":"finished"`) {
				finishedSeen = true
			}
		}
	}
	if !requestSeen || !finishedSeen {
		t.Fatalf("request=%v finished=%v", requestSeen, finishedSeen)
	}
}

func TestSSE_AlreadyFinished_Returns204(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	mux := http.NewServeMux()
	audit.RegisterRoutes(mux, mgr)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	id, _ := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:   "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
	})
	time.Sleep(150 * time.Millisecond)
	resp, _ := http.Get(srv.URL + "/api/v1/audits/" + id + "/events")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestSSE_SubscribeAfterFinalize_ClosesCleanly(t *testing.T) {
	mgr, _, cleanup := openTestManager(t)
	defer cleanup()
	mux := http.NewServeMux()
	audit.RegisterRoutes(mux, mgr)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	id, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:   "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	// Wait for run to finish AND for the prune goroutine to delete from map.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := mgr.Get(id); !ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	resp, err := http.Get(srv.URL + "/api/v1/audits/" + id + "/events")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: %d (expected 204)", resp.StatusCode)
	}
}
