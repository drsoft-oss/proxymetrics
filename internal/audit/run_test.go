package audit_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/audit"
	"github.com/drsoft-oss/proxymetrics/internal/geo"
)

type scriptedLookup struct {
	results  []geo.Result
	idx      atomic.Int32
	delay    time.Duration
	inFlight atomic.Int32
	maxSeen  atomic.Int32
}

func (s *scriptedLookup) Lookup(ctx context.Context, _ *http.Client) (geo.Result, error) {
	cur := s.inFlight.Add(1)
	for {
		mx := s.maxSeen.Load()
		if cur <= mx || s.maxSeen.CompareAndSwap(mx, cur) {
			break
		}
	}
	defer s.inFlight.Add(-1)
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return geo.Result{}, ctx.Err()
		}
	}
	i := int(s.idx.Add(1) - 1)
	if i >= len(s.results) {
		return geo.Result{}, errors.New("out of scripted results")
	}
	return s.results[i], nil
}

// seedRun inserts the audit_runs row that audit.Run.Execute will later
// UPDATE via FinaliseRun. The plan body forgot this; we add it here so the
// status assertions actually reflect the run's terminal state.
func seedRun(t *testing.T, as *audit.Store, id string, requestCount int) {
	t.Helper()
	if err := as.InsertRun(context.Background(), audit.RunSummary{
		ID:           id,
		StartedAt:    time.Now().UTC(),
		Status:       audit.StatusRunning,
		ProxyURL:     "x",
		ExpectedLat:  0,
		ExpectedLon:  0,
		ExpectedType: "residential",
		CheckLevel:   "country",
		RequestCount: requestCount,
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
}

func TestRun_HappyPath_RecordsMatches(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rh", 3)

	lookup := &scriptedLookup{results: []geo.Result{
		{IP: "1.1.1.1", Country: "RO", IsDatacenter: false, IsMobile: false, Source: "ipapi"},
		{IP: "2.2.2.2", Country: "RO", IsDatacenter: false, IsMobile: false, Source: "ipapi"},
		{IP: "3.3.3.3", Country: "DE", IsDatacenter: false, IsMobile: false, Source: "ipapi"},
	}}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rh", Spec: audit.Spec{
			ProxyURL: "http://user-country-RO:p@h:1", ExpectedCountry: "RO",
			ExpectedType: "residential", RequestCount: 3,
		},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       2,
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	r.Execute(context.Background())

	got, err := as.GetRun(context.Background(), "rh")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != audit.StatusCompleted {
		t.Fatalf("status: %v", got.Status)
	}
	if got.CompletedCount != 3 {
		t.Fatalf("completed: %d", got.CompletedCount)
	}
	if got.LocationMatchCount != 2 {
		t.Fatalf("location matches: %d", got.LocationMatchCount)
	}

	rows, err := as.ListRequests(context.Background(), "rh")
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	for _, row := range rows {
		if row.Attempts != 1 {
			t.Fatalf("seq=%d: want Attempts=1 on first-try success, got %d", row.Seq, row.Attempts)
		}
	}
}

func TestRun_RespectsConcurrency(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rc", 10)

	lookup := &scriptedLookup{
		results: make([]geo.Result, 10),
		delay:   30 * time.Millisecond,
	}
	for i := range lookup.results {
		lookup.results[i] = geo.Result{IP: "1.1.1.1", Country: "RO", Source: "ipapi"}
	}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rc", Spec: audit.Spec{ProxyURL: "x", ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 10},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       3,
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	r.Execute(context.Background())

	if max := lookup.maxSeen.Load(); max > 3 {
		t.Fatalf("max in flight exceeded: %d > 3", max)
	}
}

func TestRun_Cancellation_StopsNewDispatches(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rcan", 100)

	lookup := &scriptedLookup{
		results: make([]geo.Result, 100),
		delay:   50 * time.Millisecond,
	}
	for i := range lookup.results {
		lookup.results[i] = geo.Result{IP: "1.1.1.1", Country: "RO", Source: "ipapi"}
	}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rcan", Spec: audit.Spec{ProxyURL: "x", ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 100},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       5,
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	go func() {
		time.Sleep(60 * time.Millisecond)
		r.Cancel()
	}()
	r.Execute(context.Background())

	got, _ := as.GetRun(context.Background(), "rcan")
	if got.Status != audit.StatusCancelled {
		t.Fatalf("status: %v", got.Status)
	}
	if got.CompletedCount >= 100 {
		t.Fatalf("expected partial completion, got %d", got.CompletedCount)
	}
}

func TestRun_PublishesEvents(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "re", 2)

	lookup := &scriptedLookup{results: []geo.Result{
		{IP: "1.1.1.1", Country: "RO", Source: "ipapi"},
		{IP: "2.2.2.2", Country: "RO", Source: "ipapi"},
	}}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "re", Spec: audit.Spec{ProxyURL: "x", ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 2},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	sub, cancel := r.Subscribe()
	defer cancel()
	go r.Execute(context.Background())

	var requestEvents, finishedEvents int
	for ev := range sub {
		switch ev.Type {
		case audit.EventRequest:
			requestEvents++
		case audit.EventFinished:
			finishedEvents++
		}
	}
	if requestEvents != 2 {
		t.Fatalf("request events: %d", requestEvents)
	}
	if finishedEvents != 1 {
		t.Fatalf("finished events: %d", finishedEvents)
	}
}

type recordingLookup struct {
	mu   sync.Mutex
	urls []string
	ips  []string
	idx  atomic.Int32
}

func (rl *recordingLookup) Lookup(_ context.Context, hc *http.Client) (geo.Result, error) {
	tr, ok := hc.Transport.(*http.Transport)
	if !ok || tr.Proxy == nil {
		return geo.Result{}, errors.New("no proxy on transport")
	}
	// Build a placeholder request to get the proxy URL the client would dial.
	req, _ := http.NewRequest("GET", "http://example.invalid/", nil)
	pu, err := tr.Proxy(req)
	if err != nil {
		return geo.Result{}, err
	}
	rl.mu.Lock()
	rl.urls = append(rl.urls, pu.String())
	rl.mu.Unlock()
	i := int(rl.idx.Add(1) - 1)
	ip := "10.0.0.1"
	if i < len(rl.ips) {
		ip = rl.ips[i]
	}
	return geo.Result{IP: ip, Country: "RO", Source: "ipapi"}, nil
}

func TestRun_RotatesSessionPerRequest(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	seedRun(t, as, "rrot", 5)

	rl := &recordingLookup{ips: []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4", "5.5.5.5"}}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rrot",
		Spec: audit.Spec{
			ProxyURL:        "http://user-session-original:p@h:1",
			ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
			SessionKey: "session",
		},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            rl,
		Store:             as,
		MaxInFlight:       1, // serial — keeps URL ordering deterministic
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	r.Execute(context.Background())

	rl.mu.Lock()
	urls := append([]string(nil), rl.urls...)
	rl.mu.Unlock()

	if len(urls) != 5 {
		t.Fatalf("expected 5 urls, got %d", len(urls))
	}
	seen := map[string]struct{}{}
	for _, u := range urls {
		if !strings.Contains(u, "user-session-") {
			t.Fatalf("url missing rotated session: %q", u)
		}
		if strings.Contains(u, "user-session-original") {
			t.Fatalf("url still contains original session value: %q", u)
		}
		seen[u] = struct{}{}
	}
	if len(seen) != 5 {
		t.Fatalf("expected 5 distinct urls, got %d: %v", len(seen), urls)
	}
}

func TestRun_NoSessionKey_PreservesOriginalURL(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	seedRun(t, as, "rorig", 3)

	rl := &recordingLookup{ips: []string{"1.1.1.1", "1.1.1.1", "1.1.1.1"}}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rorig",
		Spec: audit.Spec{
			ProxyURL:        "http://user-zone-residential:p@h:1",
			ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 3,
		},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            rl,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	r.Execute(context.Background())

	rl.mu.Lock()
	defer rl.mu.Unlock()
	for _, u := range rl.urls {
		if !strings.Contains(u, "user-zone-residential") {
			t.Fatalf("url should be unchanged when no session key: %q", u)
		}
	}
}

// flakyLookup fails the first `failuresBefore` invocations with errFn(), then
// returns successResults in order. Records the proxy_url passed to each call
// (via the *http.Client's Transport.Proxy func) so tests can assert that
// session rotation produced fresh values per attempt.
type flakyLookup struct {
	failuresBefore int
	successResults []geo.Result
	calls          atomic.Int32
	mu             sync.Mutex
	proxyURLs      []string
}

func (f *flakyLookup) Lookup(_ context.Context, hc *http.Client) (geo.Result, error) {
	n := int(f.calls.Add(1))
	// Best-effort capture of the proxy URL the run dialed with.
	if hc != nil && hc.Transport != nil {
		if tr, ok := hc.Transport.(*http.Transport); ok && tr.Proxy != nil {
			if u, err := tr.Proxy(nil); err == nil && u != nil {
				f.mu.Lock()
				f.proxyURLs = append(f.proxyURLs, u.String())
				f.mu.Unlock()
			}
		}
	}
	if n <= f.failuresBefore {
		return geo.Result{}, errors.New("transient: attempt " + strings.Repeat("x", n))
	}
	idx := n - f.failuresBefore - 1
	if idx >= len(f.successResults) {
		return geo.Result{}, errors.New("out of scripted results")
	}
	return f.successResults[idx], nil
}

func TestRun_Retry_FlakyOnceThenSucceeds(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rt", 1)

	lookup := &flakyLookup{
		failuresBefore: 1,
		successResults: []geo.Result{{IP: "1.1.1.1", Country: "RO", Source: "ipapi"}},
	}
	r, err := audit.NewRun(audit.RunConfig{
		ID: "rt",
		Spec: audit.Spec{
			ProxyURL: "http://u:p@h:1", ExpectedCountry: "RO",
			ExpectedType: "residential", RequestCount: 1,
		},
		Expected:          audit.Expected{Country: "RO", Type: "residential", CheckLevel: "country"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new run: %v", err)
	}
	r.Execute(context.Background())

	rows, err := as.ListRequests(context.Background(), "rt")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Error != "" {
		t.Fatalf("want no error, got %q", rows[0].Error)
	}
	if rows[0].Attempts != 2 {
		t.Fatalf("want Attempts=2, got %d", rows[0].Attempts)
	}
}

func TestRun_Retry_AlwaysFailsRecordsError(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "ra", 1)

	lookup := &flakyLookup{failuresBefore: 99} // never succeeds
	r, err := audit.NewRun(audit.RunConfig{
		ID: "ra",
		Spec: audit.Spec{
			ProxyURL: "http://u:p@h:1", ExpectedCountry: "RO",
			ExpectedType: "residential", RequestCount: 1,
		},
		Expected:          audit.Expected{Country: "RO", Type: "residential", CheckLevel: "country"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new run: %v", err)
	}
	r.Execute(context.Background())

	rows, _ := as.ListRequests(context.Background(), "ra")
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Error == "" {
		t.Fatalf("want error, got none")
	}
	if rows[0].Attempts != 3 {
		t.Fatalf("want Attempts=3, got %d", rows[0].Attempts)
	}
	if int(lookup.calls.Load()) != 3 {
		t.Fatalf("want 3 lookup calls, got %d", lookup.calls.Load())
	}
}

func TestRun_Retry_DeterministicURLParseErrorNotRetried(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rd", 1)

	lookup := &scriptedLookup{}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rd",
		Spec: audit.Spec{
			// Trailing percent makes url.Parse fail.
			ProxyURL: "http://u:p@h:1/%zz", ExpectedCountry: "RO",
			ExpectedType: "residential", RequestCount: 1,
		},
		Expected:          audit.Expected{Country: "RO", Type: "residential", CheckLevel: "country"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new run: %v", err)
	}
	r.Execute(context.Background())

	rows, _ := as.ListRequests(context.Background(), "rd")
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Error == "" || rows[0].Attempts != 1 {
		t.Fatalf("want Attempts=1 with error, got Attempts=%d Error=%q", rows[0].Attempts, rows[0].Error)
	}
	if int(lookup.idx.Load()) != 0 {
		t.Fatalf("Lookup should not have been called, idx=%d", lookup.idx.Load())
	}
}

func TestRun_Retry_FreshSessionPerAttempt(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rs", 1)

	// Force three attempts: fail twice, then succeed. flakyLookup also
	// records the proxy URL each attempt dialed with.
	lookup := &flakyLookup{
		failuresBefore: 2,
		successResults: []geo.Result{{IP: "1.1.1.1", Country: "RO", Source: "ipapi"}},
	}
	r, err := audit.NewRun(audit.RunConfig{
		ID: "rs",
		Spec: audit.Spec{
			ProxyURL:        "http://user-session-INIT:p@h:1",
			ExpectedCountry: "RO",
			ExpectedType:    "residential",
			RequestCount:    1,
			SessionKey:      "session",
		},
		Expected:          audit.Expected{Country: "RO", Type: "residential", CheckLevel: "country"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new run: %v", err)
	}
	r.Execute(context.Background())

	lookup.mu.Lock()
	defer lookup.mu.Unlock()
	if len(lookup.proxyURLs) != 3 {
		t.Fatalf("want 3 proxy URLs captured, got %d (%v)", len(lookup.proxyURLs), lookup.proxyURLs)
	}
	a, b, c := lookup.proxyURLs[0], lookup.proxyURLs[1], lookup.proxyURLs[2]
	if a == b || b == c || a == c {
		t.Fatalf("want three distinct rotated URLs, got: %q %q %q", a, b, c)
	}
}

func TestRun_Retry_CancelDuringBackoffBailsCleanly(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()

	seedRun(t, as, "rc", 1)

	lookup := &flakyLookup{failuresBefore: 99}
	r, err := audit.NewRun(audit.RunConfig{
		ID: "rc",
		Spec: audit.Spec{
			ProxyURL: "http://u:p@h:1", ExpectedCountry: "RO",
			ExpectedType: "residential", RequestCount: 1,
		},
		Expected:          audit.Expected{Country: "RO", Type: "residential", CheckLevel: "country"},
		Lookup:            lookup,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new run: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.Execute(ctx)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not finalise after cancel")
	}

	rows, _ := as.ListRequests(context.Background(), "rc")
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Error == "" {
		t.Fatalf("want error after cancel, got none")
	}
	if rows[0].Attempts < 1 || rows[0].Attempts > maxAttemptsForTest {
		t.Fatalf("Attempts out of range: %d", rows[0].Attempts)
	}
}

const maxAttemptsForTest = 3

func TestRun_TalliesUniqueIPs(t *testing.T) {
	as, _, cleanup := openTestStore(t)
	defer cleanup()
	seedRun(t, as, "rip", 5)

	// Three distinct IPs across 5 requests.
	rl := &recordingLookup{ips: []string{"1.1.1.1", "2.2.2.2", "1.1.1.1", "3.3.3.3", "2.2.2.2"}}

	r, err := audit.NewRun(audit.RunConfig{
		ID: "rip",
		Spec: audit.Spec{
			ProxyURL:        "http://user-session-x:p@h:1",
			ExpectedCountry: "RO", ExpectedType: "residential", RequestCount: 5,
			SessionKey: "session",
		},
		Expected:          audit.Expected{Country: "RO", CheckLevel: "country", Type: "residential"},
		Lookup:            rl,
		Store:             as,
		MaxInFlight:       1,
		PerRequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	r.Execute(context.Background())

	got, err := as.GetRun(context.Background(), "rip")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UniqueIPCount == nil {
		t.Fatal("unique_ip_count nil")
	}
	if *got.UniqueIPCount != 3 {
		t.Fatalf("unique_ip_count = %d want 3", *got.UniqueIPCount)
	}
}
