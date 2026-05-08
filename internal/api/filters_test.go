package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/api"
)

func TestParseEventFilter_Defaults(t *testing.T) {
	v := url.Values{}
	f, err := api.ParseEventFilter(v, time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if f.Limit != 50 {
		t.Fatalf("default limit: %d", f.Limit)
	}
	if f.From.IsZero() || f.To.IsZero() {
		t.Fatalf("from/to: %v / %v", f.From, f.To)
	}
	if f.To.Sub(f.From) != 24*time.Hour {
		t.Fatalf("default window: %v", f.To.Sub(f.From))
	}
}

func TestParseEventFilter_AllParams(t *testing.T) {
	v := url.Values{}
	v.Set("from", "2026-04-01T00:00:00Z")
	v.Set("to", "2026-04-29T23:59:59Z")
	v["profile_id"] = []string{"p1", "p2"}
	v["vendor"] = []string{"brightdata"}
	v["status_class"] = []string{"4xx", "5xx"}
	v["status_code"] = []string{"403", "429"}
	v.Set("target_host", "example.com")
	v.Set("limit", "100")
	v.Set("offset", "200")
	v.Set("sort", "ts")
	v.Set("order", "asc")

	f, err := api.ParseEventFilter(v, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.ProfileIDs) != 2 || f.ProfileIDs[0] != "p1" {
		t.Fatalf("profile_ids: %v", f.ProfileIDs)
	}
	if len(f.StatusCodes) != 2 || f.StatusCodes[0] != 403 {
		t.Fatalf("status_codes: %v", f.StatusCodes)
	}
	if f.Limit != 100 || f.Offset != 200 {
		t.Fatalf("paging: %d/%d", f.Limit, f.Offset)
	}
	if f.Sort != "ts" || f.Order != "asc" {
		t.Fatalf("sort: %q %q", f.Sort, f.Order)
	}
}

func TestParseEventFilter_ClampLimit(t *testing.T) {
	v := url.Values{}
	v.Set("limit", "999999")
	f, err := api.ParseEventFilter(v, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if f.Limit != 1000 {
		t.Fatalf("clamp: %d", f.Limit)
	}
}

func TestParseEventFilter_BadTime(t *testing.T) {
	v := url.Values{}
	v.Set("from", "yesterday")
	_, err := api.ParseEventFilter(v, time.Now().UTC())
	if err == nil {
		t.Fatal("want error")
	}
}

func TestParseEventFilter_BadStatusCode(t *testing.T) {
	v := url.Values{}
	v["status_code"] = []string{"abc"}
	_, err := api.ParseEventFilter(v, time.Now().UTC())
	if err == nil {
		t.Fatal("want error")
	}
}

func TestParseRollupFilter_Level(t *testing.T) {
	v := url.Values{}
	f, err := api.ParseRollupFilter(v, "1hour", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if f.Level != "1hour" {
		t.Fatalf("level: %q", f.Level)
	}
}

func TestParseRollupFilter_BadLevel(t *testing.T) {
	v := url.Values{}
	_, err := api.ParseRollupFilter(v, "bogus", time.Now().UTC())
	if err == nil {
		t.Fatal("want error")
	}
}
