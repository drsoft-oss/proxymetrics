package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func TestCaptchas_EmptyRange(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/captchas")
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
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rows) != 0 {
		t.Fatalf("rows: %v", body.Rows)
	}
}

func TestCaptchas_AggregatesByVendorType(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.CreateProfile(ctx, store.Profile{
		ID: "p1", Label: "L", Vendor: "brightdata", Type: "residential",
		UpstreamURL: "u", Currency: "USD",
	})
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "r1", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
		{TS: now, RequestID: "r2", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
		{TS: now, RequestID: "r3", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "turnstile"},
		{TS: now, RequestID: "r4", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: ""}, // ignored
	})

	resp, err := http.Get(srv.URL + "/api/v1/captchas")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Rows []struct {
			Vendor string           `json:"vendor"`
			Type   string           `json:"type"`
			Total  int64            `json:"total"`
			ByKind map[string]int64 `json:"by_kind"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rows) != 1 {
		t.Fatalf("got %d rows: %+v", len(body.Rows), body.Rows)
	}
	row := body.Rows[0]
	if row.Total != 3 {
		t.Fatalf("total = %d, want 3", row.Total)
	}
	if row.ByKind["recaptcha"] != 2 || row.ByKind["turnstile"] != 1 {
		t.Fatalf("by_kind = %+v", row.ByKind)
	}
}

func TestCaptchas_DistributionFlat(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "r1", ProfileID: "p1", Vendor: "v", Type: "t",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
		{TS: now, RequestID: "r2", ProfileID: "p1", Vendor: "v", Type: "t",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "hcaptcha"},
	})

	resp, err := http.Get(srv.URL + "/api/v1/captchas/distribution")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Items []struct {
			Kind     string `json:"kind"`
			Requests int64  `json:"requests"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("items: %+v", body.Items)
	}
}

func TestCaptchas_DetailDrill(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "r1", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			TargetHost: "shop.example.com", StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
		{TS: now, RequestID: "r2", ProfileID: "p1", Vendor: "brightdata", Type: "residential",
			TargetHost: "shop.example.com", StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
		{TS: now, RequestID: "r3", ProfileID: "p1", Vendor: "oxylabs", Type: "datacenter",
			TargetHost: "other.example.com", StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
	})

	resp, err := http.Get(srv.URL + "/api/v1/captchas/recaptcha")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Kind         string           `json:"kind"`
		Requests     int64            `json:"requests"`
		TopProviders []map[string]any `json:"top_providers"`
		TopTargets   []map[string]any `json:"top_targets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Kind != "recaptcha" || body.Requests != 3 {
		t.Fatalf("body: %+v", body)
	}
	if len(body.TopProviders) == 0 {
		t.Fatalf("expected providers")
	}
	if len(body.TopTargets) == 0 {
		t.Fatalf("expected targets")
	}
}

func TestCaptchas_DetailUnknownKind(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/captchas/notarealkind")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status: %d, want 400", resp.StatusCode)
	}
}

func TestCaptchas_FilterByCaptchaKind(t *testing.T) {
	srv, s := newTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()
	_ = s.WriteEvents(ctx, []store.Event{
		{TS: now, RequestID: "r1", ProfileID: "p1", Vendor: "v", Type: "t",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "recaptcha"},
		{TS: now, RequestID: "r2", ProfileID: "p1", Vendor: "v", Type: "t",
			StatusCode: 200, StatusClass: "2xx", CaptchaKind: "turnstile"},
	})
	resp, err := http.Get(srv.URL + "/api/v1/captchas?captcha_kind=recaptcha")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Rows []struct {
			Total  int64            `json:"total"`
			ByKind map[string]int64 `json:"by_kind"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rows) != 1 || body.Rows[0].Total != 1 || body.Rows[0].ByKind["turnstile"] != 0 {
		t.Fatalf("filter not applied: %+v", body.Rows)
	}
}
