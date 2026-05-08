package profile_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anonymous-proxies/proxymetrics/internal/profile"
	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

func newRegistry(t *testing.T) *profile.Registry {
	t.Helper()
	r, err := profile.NewRegistry(context.Background(), newMemStore())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestHTTP_ListAndShow(t *testing.T) {
	s := newMemStore()
	s.profiles["p1"] = store.Profile{
		ID: "p1", Label: "L", Vendor: "custom", Type: "residential",
		UpstreamURL: "http://u", Currency: "USD",
	}
	r, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	profile.RegisterHTTP(mux, r)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list status: %d", rr.Code)
	}
	var list []store.Profile
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("list len: %d", len(list))
	}

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profiles/p1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("show: %d", rr.Code)
	}
	var got store.Profile
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "p1" || got.Label != "L" {
		t.Fatalf("show body: %+v", got)
	}
}

func TestHTTP_ShowMissingReturns404(t *testing.T) {
	r := newRegistry(t)
	mux := http.NewServeMux()
	profile.RegisterHTTP(mux, r)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profiles/nope", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: %d", rr.Code)
	}
}

func TestHTTP_ListWithoutUsageFlag_UnchangedShape(t *testing.T) {
	r := newRegistry(t)
	mux := http.NewServeMux()
	profile.RegisterHTTP(mux, r)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var list []store.Profile
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least the autocreated default profile")
	}
}

func TestHTTP_ListWithUsage_ReturnsUsageFields(t *testing.T) {
	src := newMemStore()
	src.usage = []store.ProfileUsage{{ProfileID: "default", Requests: 42, SpendUSD: 5.25}}
	r, err := profile.NewRegistry(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	profile.RegisterHTTP(mux, r)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profiles?with=usage", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}

	var list []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var dflt map[string]any
	for _, row := range list {
		if row["id"] == "default" {
			dflt = row
			break
		}
	}
	if dflt == nil {
		t.Fatalf("default row missing: %+v", list)
	}
	if got, _ := dflt["requests"].(float64); got != 42 {
		t.Errorf("requests: got %v want 42", dflt["requests"])
	}
	if got, _ := dflt["spend_usd"].(float64); got != 5.25 {
		t.Errorf("spend_usd: got %v want 5.25", dflt["spend_usd"])
	}
}

func TestHTTP_ListWithUsage_RejectsBadWindow(t *testing.T) {
	r := newRegistry(t)
	mux := http.NewServeMux()
	profile.RegisterHTTP(mux, r)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/profiles?with=usage&window=banana", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
}

// Profiles are observed-only — POST/PATCH/DELETE must reject with 405.
func TestHTTP_RejectsMutations(t *testing.T) {
	r := newRegistry(t)
	mux := http.NewServeMux()
	profile.RegisterHTTP(mux, r)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/profiles"},
		{http.MethodPatch, "/api/v1/profiles/default"},
		{http.MethodDelete, "/api/v1/profiles/default"},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(c.method, c.path, nil))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status %d, want 405", c.method, c.path, rr.Code)
		}
	}
}
