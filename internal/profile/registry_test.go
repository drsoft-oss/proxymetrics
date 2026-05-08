package profile_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/profile"
	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

type memStore struct {
	mu          sync.Mutex
	profiles    map[string]store.Profile
	usage       []store.ProfileUsage
	upsertCalls int
}

func newMemStore() *memStore { return &memStore{profiles: map[string]store.Profile{}} }

func (m *memStore) ListProfiles(ctx context.Context) ([]store.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]store.Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		out = append(out, p)
	}
	return out, nil
}

func (m *memStore) CreateProfile(ctx context.Context, p store.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[p.ID]; ok {
		return store.ErrConflict
	}
	m.profiles[p.ID] = p
	return nil
}

func (m *memStore) ListProfilesWithUsage(ctx context.Context, window time.Duration) ([]store.ProfileUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]store.ProfileUsage, len(m.usage))
	copy(out, m.usage)
	return out, nil
}

func (m *memStore) UpsertProfileObserved(ctx context.Context, p store.Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertCalls++
	if _, ok := m.profiles[p.ID]; ok {
		return nil
	}
	m.profiles[p.ID] = p
	return nil
}

func TestRegistry_AutocreatesDefault(t *testing.T) {
	s := newMemStore()
	r, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := r.Lookup("default")
	if !ok {
		t.Fatal("default missing")
	}
	if got.Vendor != "custom" {
		t.Fatalf("default vendor: %q", got.Vendor)
	}
}

func TestRegistry_LookupAndReloadAfterUpsert(t *testing.T) {
	s := newMemStore()
	r, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.UpsertObserved(context.Background(), "brightdata", "residential"); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Lookup("brightdata-residential"); !ok {
		t.Fatal("expected brightdata-residential visible after UpsertObserved")
	}
}

func TestRegistry_UpsertObserved_InsertsThenIdempotent(t *testing.T) {
	s := newMemStore()
	r, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.upsertCalls = 0 // reset whatever bootstrap did
	s.mu.Unlock()

	if err := r.UpsertObserved(context.Background(), "brightdata", "residential"); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	got, ok := r.Lookup("brightdata-residential")
	if !ok {
		t.Fatal("expected synthetic profile to be in cache after first upsert")
	}
	if got.Vendor != "brightdata" || got.Type != "residential" {
		t.Errorf("synthesized row wrong: %+v", got)
	}
	if got.Label != "Brightdata Residential" {
		t.Errorf("synthesized label wrong: %q", got.Label)
	}

	// Repeated calls for the same combo must be cheap — no second store hit.
	for i := range 3 {
		if err := r.UpsertObserved(context.Background(), "brightdata", "residential"); err != nil {
			t.Fatalf("repeat upsert #%d: %v", i, err)
		}
	}
	s.mu.Lock()
	calls := s.upsertCalls
	s.mu.Unlock()
	if calls != 1 {
		t.Errorf("expected exactly 1 store write across 4 UpsertObserved calls, got %d", calls)
	}
}

func TestRegistry_UpsertObserved_EmptyTagsIsNoop(t *testing.T) {
	s := newMemStore()
	r, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	before := len(s.profiles)
	if err := r.UpsertObserved(context.Background(), "", ""); err != nil {
		t.Fatalf("upsert empty: %v", err)
	}
	after := len(s.profiles)
	if after != before {
		t.Errorf("empty (provider,type) should be a no-op: profiles before=%d after=%d", before, after)
	}
}

func TestRegistry_ListWithUsage(t *testing.T) {
	s := newMemStore()
	// Pre-populate the in-memory store with two profiles before the registry boots.
	s.profiles["p1"] = store.Profile{ID: "p1", Label: "L1", Vendor: "custom", Type: "residential", UpstreamURL: "http://u1", Currency: "USD"}
	s.profiles["p2"] = store.Profile{ID: "p2", Label: "L2", Vendor: "custom", Type: "residential", UpstreamURL: "http://u2", Currency: "USD"}
	// Seed usage only for p1.
	s.usage = []store.ProfileUsage{{ProfileID: "p1", Requests: 7, SpendUSD: 3.5}}

	r, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := r.ListWithUsage(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	byID := map[string]profile.ProfileWithUsage{}
	for _, row := range rows {
		byID[row.Profile.ID] = row
	}

	p1, ok := byID["p1"]
	if !ok || p1.Requests != 7 || p1.SpendUSD != 3.5 {
		t.Fatalf("p1 row wrong: %+v", p1)
	}
	p2, ok := byID["p2"]
	if !ok {
		t.Fatal("p2 missing — registry should LEFT-JOIN to fill 0/0")
	}
	if p2.Requests != 0 || p2.SpendUSD != 0 {
		t.Errorf("p2 should be 0/0, got %+v", p2)
	}

	// The autocreated default should also be present.
	if _, ok := byID["default"]; !ok {
		t.Error("default profile should be in result")
	}
}
