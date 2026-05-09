package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/api"
	"github.com/drsoft-oss/proxymetrics/internal/broadcaster"
	"github.com/drsoft-oss/proxymetrics/internal/store"
	"github.com/drsoft-oss/proxymetrics/internal/store/sqlite"
)

func newTestServer(t *testing.T) (*httptest.Server, *sqlite.Store) {
	t.Helper()
	s, err := sqlite.Open(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	api.Register(mux, s, broadcaster.New(), "test-secret")
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		s.Close()
	})
	return srv, s
}

func TestEvents_List(t *testing.T) {
	srv, s := newTestServer(t)

	_ = s.CreateProfile(context.Background(), store.Profile{
		ID: "p1", Label: "L", Vendor: "custom", Type: "residential",
		UpstreamURL: "u", Currency: "USD",
	})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "example.com", StatusCode: 200, StatusClass: "2xx",
			BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 0.001,
			Team: "growth", Project: "scrape"},
	})

	resp, err := http.Get(srv.URL + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var body struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total_estimate"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 1 || body.Total != 1 {
		t.Fatalf("body: %+v", body)
	}
}

func TestEvents_BadFilter(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/events?from=yesterday")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestEventDetail_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/events/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestEventDetail_Found(t *testing.T) {
	srv, s := newTestServer(t)
	_ = s.CreateProfile(context.Background(), store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "example.com", StatusCode: 200, StatusClass: "2xx",
			BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})
	resp, err := http.Get(srv.URL + "/api/v1/events/r1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestRollups_QueryByLevel(t *testing.T) {
	srv, s := newTestServer(t)
	bucket := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	_ = s.WriteRollups(context.Background(), "1hour", []store.RollupRow{
		{TSBucket: bucket, ProfileID: "p1", Vendor: "v", Type: "residential", StatusClass: "2xx",
			RequestCount: 10, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0,
			SuccessCount: 10, FailureCount: 0},
	})

	resp, err := http.Get(srv.URL + "/api/v1/rollups/1hour?from=2026-04-29T00:00:00Z&to=2026-04-29T23:59:59Z")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 1 {
		t.Fatalf("items: %d", len(body.Items))
	}
}

func TestRollups_BadLevel(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/rollups/bogus")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestProviders_List(t *testing.T) {
	srv, s := newTestServer(t)
	_ = s.CreateProfile(context.Background(), store.Profile{ID: "p1", Label: "Bright Data", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			StatusClass: "2xx", BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 1.0},
	})

	resp, err := http.Get(srv.URL + "/api/v1/providers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) == 0 {
		t.Fatal("no items")
	}
}

func TestProviderDetail_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/providers/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestOverview_FromEvents(t *testing.T) {
	srv, s := newTestServer(t)
	_ = s.CreateProfile(context.Background(), store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "2xx", BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 1.0},
		{TS: time.Now().UTC(), RequestID: "r2", ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "4xx", BytesIn: 500, BytesOut: 50, LatencyMS: 30, CostUSD: 0.5},
	})

	resp, err := http.Get(srv.URL + "/api/v1/overview")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var body struct {
		SpendUSD     float64 `json:"spend_usd_total"`
		WastedUSD    float64 `json:"wasted_usd_total"`
		WastedPct    float64 `json:"wasted_pct_of_spend"`
		BytesTotal   int64   `json:"bytes_total"`
		RequestCount int64   `json:"request_count"`
		SuccessRate  float64 `json:"success_rate"`
		Source       string  `json:"source"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.RequestCount != 2 {
		t.Fatalf("count: %d", body.RequestCount)
	}
	if body.SpendUSD != 1.5 {
		t.Fatalf("spend: %v", body.SpendUSD)
	}
	if body.WastedUSD != 0.5 {
		t.Fatalf("wasted: %v", body.WastedUSD)
	}
	if body.Source != "events" {
		t.Fatalf("source: %q", body.Source)
	}
}

func TestTargets_List(t *testing.T) {
	srv, s := newTestServer(t)
	_ = s.CreateProfile(context.Background(), store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "amazon.com", StatusClass: "2xx", BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 0.1},
		{TS: time.Now().UTC(), RequestID: "r2", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "ebay.com", StatusClass: "2xx", BytesIn: 1500, BytesOut: 150, LatencyMS: 60, CostUSD: 0.15},
	})

	resp, err := http.Get(srv.URL + "/api/v1/targets")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 2 {
		t.Fatalf("items: %d", len(body.Items))
	}
}

func TestTargetDetail_Found(t *testing.T) {
	srv, s := newTestServer(t)
	_ = s.CreateProfile(context.Background(), store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			TargetHost: "amazon.com", StatusClass: "2xx", BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 0.1},
	})

	resp, err := http.Get(srv.URL + "/api/v1/targets/amazon.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestStatusCodes(t *testing.T) {
	srv, s := newTestServer(t)
	_ = s.CreateProfile(context.Background(), store.Profile{ID: "p1", Label: "L", Vendor: "custom", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(context.Background(), []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "2xx", BytesIn: 1000, BytesOut: 100, LatencyMS: 50, CostUSD: 1.0},
		{TS: time.Now().UTC(), RequestID: "r2", ProfileID: "p1", Vendor: "custom", Type: "residential",
			StatusClass: "4xx", BytesIn: 500, BytesOut: 50, LatencyMS: 30, CostUSD: 0.5},
	})

	resp, err := http.Get(srv.URL + "/api/v1/status-codes")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Rows []map[string]any `json:"rows"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Rows) == 0 {
		t.Fatal("no rows")
	}
}

func TestSessionsActive_Empty(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/sessions/active")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body []any
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body) != 0 {
		t.Fatalf("expected []: %v", body)
	}
}

func TestStatusCodes_Distribution(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "smartproxy", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: time.Now().UTC(), RequestID: "a", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", StatusCode: 200, StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: time.Now().UTC(), RequestID: "b", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
	})

	resp, err := http.Get(srv.URL + "/api/v1/status-codes/distribution")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Items  []map[string]any `json:"items"`
		Source string           `json:"source"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 2 {
		t.Fatalf("items: %d", len(body.Items))
	}
	if body.Source != "events" {
		t.Fatalf("source: %q", body.Source)
	}
}

func TestStatusCodes_DetailFound(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "smartproxy", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: time.Now().UTC(), RequestID: "a", ProfileID: "p1", Vendor: "smartproxy", Type: "residential", TargetHost: "x.com", StatusCode: 429, StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
	})

	resp, err := http.Get(srv.URL + "/api/v1/status-codes/429")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Code         int              `json:"code"`
		Class        string           `json:"class"`
		TopProviders []map[string]any `json:"top_providers"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Code != 429 || body.Class != "4xx" {
		t.Fatalf("body: %+v", body)
	}
	if len(body.TopProviders) != 1 {
		t.Fatalf("top_providers: %v", body.TopProviders)
	}
}

func TestStatusCodes_DetailNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/status-codes/418")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestStatusCodes_DetailBadCode(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/status-codes/abc")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestProviders_StatusMixAndSparkline(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})

	now := time.Now().UTC()
	// Three classes of events for status_mix.
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "a", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: now, RequestID: "b", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
		{TS: now, RequestID: "c", ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusClass: "5xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
	})
	// Rollup row in the trailing 24h to feed requests_by_hour.
	_ = s.WriteRollups(ctx, "1hour", []store.RollupRow{
		{TSBucket: now.Truncate(time.Hour), ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusClass: "2xx", RequestCount: 17, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 17, FailureCount: 0},
	})

	resp, err := http.Get(srv.URL + "/api/v1/providers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 1 {
		t.Fatalf("items: %d", len(body.Items))
	}
	row := body.Items[0]

	mix, ok := row["status_mix"].(map[string]any)
	if !ok {
		t.Fatalf("status_mix missing or wrong shape: %+v", row["status_mix"])
	}
	for _, k := range []string{"class2xx", "class3xx", "class4xx", "class5xx"} {
		if _, ok := mix[k]; !ok {
			t.Fatalf("status_mix missing %q: %+v", k, mix)
		}
	}
	if int(mix["class2xx"].(float64)) != 1 || int(mix["class4xx"].(float64)) != 1 || int(mix["class5xx"].(float64)) != 1 {
		t.Fatalf("status_mix counts: %+v", mix)
	}

	rbh, ok := row["requests_by_hour"].([]any)
	if !ok {
		t.Fatalf("requests_by_hour missing or wrong shape: %+v", row["requests_by_hour"])
	}
	if len(rbh) != 24 {
		t.Fatalf("requests_by_hour length: %d, want 24", len(rbh))
	}
	// Last bucket = current hour, must equal 17.
	if int(rbh[23].(float64)) != 17 {
		t.Fatalf("requests_by_hour[23]: %v, want 17", rbh[23])
	}
}

func TestTargets_StatusMixAndSparkline(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})

	now := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "a", ProfileID: "p1", Vendor: "brightdata", Type: "residential", TargetHost: "x.com", StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 1.0},
		{TS: now, RequestID: "b", ProfileID: "p1", Vendor: "brightdata", Type: "residential", TargetHost: "x.com", StatusClass: "4xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0.5},
	})
	_ = s.WriteRollups(ctx, "1hour", []store.RollupRow{
		{TSBucket: now.Truncate(time.Hour), ProfileID: "p1", Vendor: "brightdata", Type: "residential", StatusClass: "2xx", TargetHost: "x.com", RequestCount: 9, BytesInTotal: 1, BytesOutTotal: 1, LatencyMSAvg: 1, CostUSDTotal: 0, SuccessCount: 9, FailureCount: 0},
	})

	resp, err := http.Get(srv.URL + "/api/v1/targets")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Items []map[string]any `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 1 {
		t.Fatalf("items: %d", len(body.Items))
	}
	row := body.Items[0]

	mix, ok := row["status_mix"].(map[string]any)
	if !ok {
		t.Fatalf("status_mix missing: %+v", row)
	}
	if int(mix["class2xx"].(float64)) != 1 || int(mix["class4xx"].(float64)) != 1 {
		t.Fatalf("status_mix: %+v", mix)
	}

	rbh, ok := row["requests_by_hour"].([]any)
	if !ok {
		t.Fatalf("requests_by_hour missing: %+v", row)
	}
	if len(rbh) != 24 || int(rbh[23].(float64)) != 9 {
		t.Fatalf("requests_by_hour: %+v", rbh)
	}
}

func TestDimensions_FixedAndDistinct(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "Bright Data", Vendor: "brightdata", Type: "residential", UpstreamURL: "u", Currency: "USD"})
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: time.Now().UTC(), RequestID: "r1", ProfileID: "p1", Vendor: "brightdata", Type: "residential", Region: "us", Team: "growth", Project: "amazon", StatusClass: "2xx", BytesIn: 1, BytesOut: 1, LatencyMS: 1, CostUSD: 0},
	})

	resp, err := http.Get(srv.URL + "/api/v1/dimensions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var body struct {
		Vendor      []string            `json:"vendor"`
		Type        []string            `json:"type"`
		Region      []string            `json:"region"`
		Team        []string            `json:"team"`
		Project     []string            `json:"project"`
		StatusClass []string            `json:"status_class"`
		ProfileID   []map[string]string `json:"profile_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.StatusClass) != 4 {
		t.Fatalf("status_class: %v", body.StatusClass)
	}
	if len(body.Vendor) != 1 || body.Vendor[0] != "brightdata" {
		t.Fatalf("vendor: %v", body.Vendor)
	}
	if len(body.ProfileID) != 1 || body.ProfileID[0]["id"] != "p1" || body.ProfileID[0]["name"] != "Bright Data" {
		t.Fatalf("profile_id: %v", body.ProfileID)
	}
}
