package audit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/audit"
)

func newServerWithMgr(t *testing.T) (string, *audit.Manager, func()) {
	t.Helper()
	mgr, _, cleanup := openTestManager(t)
	mux := http.NewServeMux()
	audit.RegisterRoutes(mux, mgr)
	srv := httptest.NewServer(mux)
	return srv.URL, mgr, func() { srv.Close(); cleanup() }
}

func TestHTTP_Post_BadRequest(t *testing.T) {
	url, _, cleanup := newServerWithMgr(t)
	defer cleanup()

	// Missing expected_country.
	body := bytes.NewBufferString(`{"proxy_url":"http://user-country-RO:p@h:1","expected_type":"residential","request_count":1}`)
	resp, err := http.Post(url+"/api/v1/audits", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestHTTP_Post_StartsRun(t *testing.T) {
	url, _, cleanup := newServerWithMgr(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"proxy_url":"http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1","expected_country":"RO","expected_state":"Bucharest","expected_city":"Bucharest","expected_type":"residential","request_count":1}`)
	resp, err := http.Post(url+"/api/v1/audits", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.ID == "" || out.Status != "running" {
		t.Fatalf("got %+v", out)
	}
}

func TestHTTP_Get_NotFound(t *testing.T) {
	url, _, cleanup := newServerWithMgr(t)
	defer cleanup()
	resp, _ := http.Get(url + "/api/v1/audits/nope")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestHTTP_Cancel_Idempotent(t *testing.T) {
	url, _, cleanup := newServerWithMgr(t)
	defer cleanup()

	resp, _ := http.Post(url+"/api/v1/audits/whatever/cancel", "application/json", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestHTTP_List_ReturnsRuns(t *testing.T) {
	url, mgr, cleanup := newServerWithMgr(t)
	defer cleanup()

	_, _ = mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO-state-Bucharest-city-Bucharest-type-residential:p@h:1",
		ExpectedCountry: "RO",
		ExpectedState:   "Bucharest",
		ExpectedCity:    "Bucharest",
		ExpectedType:    "residential",
		RequestCount:    1,
	})
	time.Sleep(80 * time.Millisecond)

	resp, _ := http.Get(url + "/api/v1/audits")
	var out []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out) == 0 {
		t.Fatal("expected at least one run")
	}
}

func TestHTTP_PostAccepts_SessionKey(t *testing.T) {
	url, _, cleanup := newServerWithMgr(t)
	defer cleanup()

	body := strings.NewReader(`{
		"proxy_url":"http://user-session-old:p@h:1",
		"expected_country":"RO",
		"expected_type":"residential",
		"request_count":5,
		"session_key":"session"
	}`)
	resp, err := http.Post(url+"/api/v1/audits", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, raw)
	}
	var out struct{ ID string }
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.ID == "" {
		t.Fatal("empty id")
	}

	// GET should return the run with session_key set in JSON.
	gresp, err := http.Get(url + "/api/v1/audits/" + out.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gresp.Body.Close()
	raw, _ := io.ReadAll(gresp.Body)
	if !strings.Contains(string(raw), `"session_key":"session"`) {
		t.Fatalf("session_key not in response: %s", raw)
	}
}

func TestHTTP_PostRejects_BadSessionKey(t *testing.T) {
	url, _, cleanup := newServerWithMgr(t)
	defer cleanup()

	body := strings.NewReader(`{
		"proxy_url":"http://user-zone-residential:p@h:1",
		"expected_country":"RO",
		"expected_type":"residential",
		"request_count":5,
		"session_key":"session"
	}`)
	resp, err := http.Post(url+"/api/v1/audits", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestHTTP_GetProviders_MergesStaticAndHistory(t *testing.T) {
	url, mgr, cleanup := newServerWithMgr(t)
	defer cleanup()

	// Seed history: one provider that overlaps the static map ("brightdata"
	// via the hostname guess) and one ad-hoc string typed by a user.
	if _, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO:p@brd.superproxy.io:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
	}); err != nil {
		t.Fatalf("seed1: %v", err)
	}
	if _, err := mgr.Start(context.Background(), audit.Spec{
		ProxyURL:        "http://user-country-RO:p@unknown.example:1",
		ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 1,
		Provider:        "my-internal-pool",
	}); err != nil {
		t.Fatalf("seed2: %v", err)
	}

	resp, err := http.Get(url + "/api/v1/audits/providers")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var out struct {
		Providers        []string          `json:"providers"`
		HostnameSuffixes map[string]string `json:"hostname_suffixes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Static names must all be present.
	for _, want := range []string{"brightdata", "databay", "oxylabs", "smartproxy", "soax", "iproyal"} {
		if !contains(out.Providers, want) {
			t.Fatalf("providers missing %q: %v", want, out.Providers)
		}
	}
	// History-only ad-hoc value must be present.
	if !contains(out.Providers, "my-internal-pool") {
		t.Fatalf("providers missing %q: %v", "my-internal-pool", out.Providers)
	}
	// Dedup: exactly one "brightdata" even though it appears in both sources.
	count := 0
	for _, p := range out.Providers {
		if p == "brightdata" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("brightdata appears %d times; want 1", count)
	}
	// Sorted (case-insensitive ASC).
	for i := 1; i < len(out.Providers); i++ {
		if strings.ToLower(out.Providers[i-1]) > strings.ToLower(out.Providers[i]) {
			t.Fatalf("providers not sorted: %v", out.Providers)
		}
	}
	// Hostname suffix map exposes the static table.
	if out.HostnameSuffixes["superproxy.io"] != "brightdata" {
		t.Fatalf("hostname_suffixes: %v", out.HostnameSuffixes)
	}
	if out.HostnameSuffixes["databay.co"] != "databay" {
		t.Fatalf("hostname_suffixes: %v", out.HostnameSuffixes)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
