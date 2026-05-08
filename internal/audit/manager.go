package audit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/drsoft-oss/proxymetrics/internal/geo"
)

// CentroidResolver is the subset of *geo.CentroidResolver Manager needs;
// allows stubs in tests.
type CentroidResolver interface {
	Resolve(ctx context.Context, country, state, city string) (geo.Centroid, error)
}

// ManagerConfig is what Manager needs at startup.
type ManagerConfig struct {
	Store             *Store
	Lookup            geo.Lookup
	Centroids         CentroidResolver
	MaxInFlight       int
	PerRequestTimeout time.Duration
	MinRequestsPerRun int
	MaxRequestsPerRun int
	// BaseCtx is a long-lived context cancelled when the server shuts down.
	// nil falls back to context.Background().
	BaseCtx context.Context
}

// Manager is the audit module's public façade.
type Manager struct {
	cfg ManagerConfig

	mu   sync.Mutex
	runs map[string]*Run
}

// NewManager constructs a Manager.
func NewManager(cfg ManagerConfig) *Manager {
	if cfg.BaseCtx == nil {
		cfg.BaseCtx = context.Background()
	}
	return &Manager{cfg: cfg, runs: map[string]*Run{}}
}

// CleanupOrphans marks stale `running` rows as `failed`. Called once at startup.
func (m *Manager) CleanupOrphans(ctx context.Context) error {
	return m.cfg.Store.MarkOrphanedRunningFailed(ctx)
}

// Start validates the spec, resolves centroid, persists the run, and launches
// the goroutine. Returns the new run id.
func (m *Manager) Start(ctx context.Context, spec Spec) (string, error) {
	if err := m.validate(spec); err != nil {
		return "", err
	}
	if spec.SessionKey != "" {
		if _, err := RotateSession(spec.ProxyURL, spec.SessionKey, "_dryrun_"); err != nil {
			return "", err
		}
	}
	// Provider resolution precedence: explicit Spec.Provider > credtag
	// `provider-X` token > static hostname-suffix guess > empty. The hostname
	// fallback exists so non-UI clients (curl/scripts) get the same labelling
	// the UI prefills automatically.
	provider := spec.Provider
	if provider == "" {
		if tag, err := ParseCredtag(spec.ProxyURL); err == nil && tag.Provider != "" {
			provider = tag.Provider
		} else if u, err := url.Parse(spec.ProxyURL); err == nil {
			provider = GuessProviderFromHost(u.Hostname())
		}
	}

	country := strings.ToUpper(spec.ExpectedCountry)
	checkLevel := deriveCheckLevel(spec)

	cen, err := m.cfg.Centroids.Resolve(ctx, country, spec.ExpectedState, spec.ExpectedCity)
	if err != nil {
		return "", fmt.Errorf(
			"could not resolve centroid for '%s, %s, %s': %w",
			spec.ExpectedCity, spec.ExpectedState, country, err,
		)
	}

	id := ulid.Make().String()
	masked := maskPassword(spec.ProxyURL)

	expected := Expected{
		Country:    country,
		State:      spec.ExpectedState,
		City:       spec.ExpectedCity,
		Lat:        cen.Lat,
		Lon:        cen.Lon,
		Type:       spec.ExpectedType,
		CheckLevel: checkLevel,
	}

	var sessionKeyPtr *string
	if spec.SessionKey != "" {
		sk := spec.SessionKey
		sessionKeyPtr = &sk
	}

	now := time.Now().UTC()
	if err := m.cfg.Store.InsertRun(ctx, RunSummary{
		ID: id, StartedAt: now, Status: StatusRunning, ProxyURL: masked,
		Provider: provider, ExpectedCountry: country,
		ExpectedState: spec.ExpectedState, ExpectedCity: spec.ExpectedCity,
		ExpectedLat: cen.Lat, ExpectedLon: cen.Lon,
		ExpectedType: spec.ExpectedType, CheckLevel: checkLevel,
		RequestCount: spec.RequestCount, SessionKey: sessionKeyPtr,
	}); err != nil {
		return "", err
	}

	r, err := NewRun(RunConfig{
		ID: id, Spec: spec, Expected: expected,
		Lookup: m.cfg.Lookup, Store: m.cfg.Store,
		MaxInFlight: m.cfg.MaxInFlight, PerRequestTimeout: m.cfg.PerRequestTimeout,
	})
	if err != nil {
		return "", err
	}
	ch, unsub := r.Subscribe()

	m.mu.Lock()
	m.runs[id] = r
	m.mu.Unlock()

	// Prune the map when the run finishes. Run.finalise publishes
	// EventFinished and then closes the subscriber channel, so this
	// goroutine exits cleanly via the range-over-closed-channel path.
	go func() {
		defer unsub()
		for ev := range ch {
			if ev.Type == EventFinished {
				m.mu.Lock()
				delete(m.runs, id)
				m.mu.Unlock()
			}
		}
	}()
	go r.Execute(m.cfg.BaseCtx)
	return id, nil
}

// Get returns the in-memory *Run if the run is still live, else (nil, false).
func (m *Manager) Get(id string) (*Run, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	return r, ok
}

// GetSummary returns the persisted run summary (works for finished runs too).
func (m *Manager) GetSummary(ctx context.Context, id string) (RunSummary, error) {
	return m.cfg.Store.GetRun(ctx, id)
}

// ListSummaries returns the most recent runs.
func (m *Manager) ListSummaries(ctx context.Context, limit, offset int) ([]RunSummary, error) {
	return m.cfg.Store.ListRuns(ctx, limit, offset)
}

// ListDistinctProviders returns the unique provider strings ever recorded
// in audit_runs, sorted case-insensitively.
func (m *Manager) ListDistinctProviders(ctx context.Context) ([]string, error) {
	return m.cfg.Store.ListDistinctProviders(ctx)
}

// ListRequests returns per-request rows for a run.
func (m *Manager) ListRequests(ctx context.Context, id string) ([]RequestRow, error) {
	return m.cfg.Store.ListRequests(ctx, id)
}

// Cancel cancels a live run; idempotent.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	r, ok := m.runs[id]
	m.mu.Unlock()
	if !ok {
		return nil // already finished or never existed
	}
	r.Cancel()
	return nil
}

func (m *Manager) validate(spec Spec) error {
	if spec.ProxyURL == "" {
		return errors.New("proxy_url: required")
	}
	if spec.ExpectedCountry == "" {
		return errors.New("expected_country: required")
	}
	switch spec.ExpectedType {
	case "residential", "mobile", "datacenter":
	default:
		return fmt.Errorf("expected_type: must be one of residential|mobile|datacenter")
	}
	if spec.RequestCount < m.cfg.MinRequestsPerRun || spec.RequestCount > m.cfg.MaxRequestsPerRun {
		return fmt.Errorf("request_count: must be in [%d, %d]", m.cfg.MinRequestsPerRun, m.cfg.MaxRequestsPerRun)
	}
	return nil
}

// deriveCheckLevel returns the deepest non-empty geo field on the spec.
// city > state > country.
func deriveCheckLevel(spec Spec) string {
	if spec.ExpectedCity != "" {
		return "city"
	}
	if spec.ExpectedState != "" {
		return "state"
	}
	return "country"
}

// maskPassword swaps the password in a URL for "***" so persisted rows don't
// leak credentials.
func maskPassword(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	if _, hasPwd := u.User.Password(); !hasPwd {
		return raw
	}
	u.User = url.UserPassword(u.User.Username(), "***")
	// url.UserPassword escapes "*" as "%2A" when the URL is serialized,
	// so undo that for the masked password placeholder.
	return strings.Replace(u.String(), "%2A%2A%2A", "***", 1)
}
