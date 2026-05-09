package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func TestListProfilesWithUsage_AggregatesAndRespectsWindow(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	// p1 has 2 in-window events + 1 outside; p2 has 1 in-window event; p3 has nothing.
	_ = s.CreateProfile(ctx, store.Profile{ID: "p1", Label: "L1", Vendor: "custom", Type: "residential", UpstreamURL: "http://u1", Currency: "USD"})
	_ = s.CreateProfile(ctx, store.Profile{ID: "p2", Label: "L2", Vendor: "custom", Type: "residential", UpstreamURL: "http://u2", Currency: "USD"})
	_ = s.CreateProfile(ctx, store.Profile{ID: "p3", Label: "L3", Vendor: "custom", Type: "residential", UpstreamURL: "http://u3", Currency: "USD"})

	now := time.Now().UTC()
	inWindow1 := now.Add(-1 * time.Hour)
	inWindow2 := now.Add(-12 * time.Hour)
	outWindow := now.Add(-25 * time.Hour) // outside trailing-24h

	if err := s.WriteEvents(ctx, []store.Event{
		{TS: inWindow1, RequestID: "r1", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 1000, LatencyMS: 10, CostUSD: 1.50},
		{TS: inWindow2, RequestID: "r2", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 2000, LatencyMS: 12, CostUSD: 2.50},
		{TS: outWindow, RequestID: "r3", ProfileID: "p1", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 3000, LatencyMS: 14, CostUSD: 9.99},
		{TS: inWindow1, RequestID: "r4", ProfileID: "p2", Vendor: "custom", Type: "residential", StatusClass: "2xx", BytesIn: 500, LatencyMS: 8, CostUSD: 0.75},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := s.ListProfilesWithUsage(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("ListProfilesWithUsage: %v", err)
	}

	byID := map[string]store.ProfileUsage{}
	for _, u := range got {
		byID[u.ProfileID] = u
	}

	p1, ok := byID["p1"]
	if !ok {
		t.Fatalf("p1 missing from result: %+v", got)
	}
	if p1.Requests != 2 {
		t.Errorf("p1 requests: got %d want 2", p1.Requests)
	}
	if p1.SpendUSD != 4.00 {
		t.Errorf("p1 spend: got %v want 4.00", p1.SpendUSD)
	}

	p2, ok := byID["p2"]
	if !ok {
		t.Fatalf("p2 missing from result: %+v", got)
	}
	if p2.Requests != 1 {
		t.Errorf("p2 requests: got %d want 1", p2.Requests)
	}
	if p2.SpendUSD != 0.75 {
		t.Errorf("p2 spend: got %v want 0.75", p2.SpendUSD)
	}

	if _, ok := byID["p3"]; ok {
		t.Errorf("p3 should not appear when it has no events in window")
	}
}

func TestListProfilesWithUsage_EmptyDB(t *testing.T) {
	s := openTemp(t)
	got, err := s.ListProfilesWithUsage(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("ListProfilesWithUsage: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %+v", got)
	}
}

func TestUpsertProfileObserved_InsertsThenIsIdempotent(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	p := store.Profile{
		ID:          "brightdata-residential",
		Label:       "Brightdata Residential",
		Vendor:      "brightdata",
		Type:        "residential",
		UpstreamURL: "http://placeholder.invalid:1/",
		Currency:    "USD",
	}

	if err := s.UpsertProfileObserved(ctx, p); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	got, err := s.GetProfile(ctx, p.ID)
	if err != nil {
		t.Fatalf("get after first upsert: %v", err)
	}
	if got.Vendor != "brightdata" || got.Type != "residential" {
		t.Fatalf("first upsert wrote wrong row: %+v", got)
	}

	// Second upsert with different label/vendor must NOT overwrite the existing
	// row — the admin path is the source of truth for editable fields, and
	// observe-from-traffic must never clobber it.
	mutated := p
	mutated.Label = "DIFFERENT"
	mutated.Vendor = "DIFFERENT"
	if err := s.UpsertProfileObserved(ctx, mutated); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got2, err := s.GetProfile(ctx, p.ID)
	if err != nil {
		t.Fatalf("get after second upsert: %v", err)
	}
	if got2.Label != "Brightdata Residential" {
		t.Errorf("upsert overwrote label: got %q", got2.Label)
	}
	if got2.Vendor != "brightdata" {
		t.Errorf("upsert overwrote vendor: got %q", got2.Vendor)
	}

	all, err := s.ListProfiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Errorf("expected exactly 1 row after idempotent upserts, got %d", len(all))
	}
}
