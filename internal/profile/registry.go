// Package profile owns the in-memory profile cache and HTTP read-only handlers.
package profile

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// DefaultID is the auto-created fallback profile.
const DefaultID = "default"

// Source is the persistence dependency Registry needs. Profiles are observed-
// only on the wire — there's no admin update/delete path, so this interface
// only carries reads, the bootstrap default-profile create, and the
// auto-discovery upsert.
type Source interface {
	ListProfiles(ctx context.Context) ([]store.Profile, error)
	CreateProfile(ctx context.Context, p store.Profile) error
	ListProfilesWithUsage(ctx context.Context, window time.Duration) ([]store.ProfileUsage, error)
	UpsertProfileObserved(ctx context.Context, p store.Profile) error
}

// Registry caches profiles in memory and reloads on every successful write.
type Registry struct {
	src Source

	mu   sync.RWMutex
	byID map[string]store.Profile
}

// NewRegistry loads from src, creates the default profile if missing, and returns
// a populated Registry.
func NewRegistry(ctx context.Context, src Source) (*Registry, error) {
	r := &Registry{src: src, byID: map[string]store.Profile{}}
	if err := r.reload(ctx); err != nil {
		return nil, err
	}
	if _, ok := r.Lookup(DefaultID); !ok {
		dflt := store.Profile{
			ID:          DefaultID,
			Label:       "Default (placeholder)",
			Vendor:      "custom",
			Type:        "residential",
			UpstreamURL: "http://placeholder.invalid:1/",
			Currency:    "USD",
		}
		if err := src.CreateProfile(ctx, dflt); err != nil {
			return nil, fmt.Errorf("autocreate default: %w", err)
		}
		if err := r.reload(ctx); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Lookup returns the cached profile or false.
func (r *Registry) Lookup(id string) (store.Profile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byID[id]
	return p, ok
}

// All returns a copy of every cached profile.
func (r *Registry) All() []store.Profile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]store.Profile, 0, len(r.byID))
	for _, p := range r.byID {
		out = append(out, p)
	}
	return out
}

// UpsertObserved is the auto-discovery entry point: when the proxy hot path
// observes a (provider, type) combo, it calls this from the event-writer
// goroutine so the registry has a row to render in the dashboard. The store
// uses INSERT ... ON CONFLICT DO NOTHING so an admin-edited row is never
// clobbered. A new row only triggers a cache reload when one was actually
// inserted (we detect via Lookup).
func (r *Registry) UpsertObserved(ctx context.Context, provider, typ string) error {
	if provider == "" && typ == "" {
		return nil
	}
	id := SyntheticID(provider, typ)
	if _, ok := r.Lookup(id); ok {
		return nil
	}
	row := store.Profile{
		ID:          id,
		Label:       SyntheticLabel(provider, typ),
		Vendor:      provider,
		Type:        typ,
		UpstreamURL: "http://placeholder.invalid:1/",
		Currency:    "USD",
	}
	if err := r.src.UpsertProfileObserved(ctx, row); err != nil {
		return err
	}
	return r.reload(ctx)
}

func (r *Registry) reload(ctx context.Context) error {
	all, err := r.src.ListProfiles(ctx)
	if err != nil {
		return fmt.Errorf("reload profiles: %w", err)
	}
	m := make(map[string]store.Profile, len(all))
	for _, p := range all {
		m[p.ID] = p
	}
	r.mu.Lock()
	r.byID = m
	r.mu.Unlock()
	return nil
}

// ProfileWithUsage pairs a registry profile row with its trailing-window
// usage aggregate (filled with 0/0 when the source returned no row).
type ProfileWithUsage struct {
	Profile  store.Profile
	Requests int64
	SpendUSD float64
}

// ListWithUsage returns every cached profile, joined with trailing-window
// request count and USD spend. Profiles with no events in the window get
// 0/0 (LEFT-JOIN behavior).
func (r *Registry) ListWithUsage(ctx context.Context, window time.Duration) ([]ProfileWithUsage, error) {
	usage, err := r.src.ListProfilesWithUsage(ctx, window)
	if err != nil {
		return nil, err
	}
	idx := make(map[string]store.ProfileUsage, len(usage))
	for _, u := range usage {
		idx[u.ProfileID] = u
	}
	all := r.All()
	out := make([]ProfileWithUsage, 0, len(all))
	for _, p := range all {
		u := idx[p.ID] // zero value when missing → 0/0
		out = append(out, ProfileWithUsage{
			Profile:  p,
			Requests: u.Requests,
			SpendUSD: u.SpendUSD,
		})
	}
	return out, nil
}
