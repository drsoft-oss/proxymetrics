package audit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/drsoft-oss/proxymetrics/internal/geo"
)

// persistTimeout bounds DuckDB writes from inside *Run so a wedged DB
// doesn't leak the run's goroutines indefinitely. Detached from the
// run's parent ctx because Cancel must not abort the terminal write.
const persistTimeout = 10 * time.Second

// maxAttempts caps the number of total Lookup attempts per request seq.
// First attempt + (maxAttempts-1) retries. Failures retried; deterministic
// errors (URL parse, session rotate) bail immediately.
const maxAttempts = 3

// retryBackoff is the fixed sleep between failed Lookup attempts.
const retryBackoff = 250 * time.Millisecond

// RunConfig is the assembled context for one *Run.
type RunConfig struct {
	ID                string
	Spec              Spec
	Expected          Expected
	Lookup            geo.Lookup
	Store             *Store
	MaxInFlight       int
	PerRequestTimeout time.Duration
}

// Run is the per-audit state machine.
type Run struct {
	cfg RunConfig

	mu          sync.Mutex
	cancelled   bool
	finalized   bool // set true under mu when closeSubscribers runs
	subscribers []chan Event

	cancelFn context.CancelFunc

	// completed-request bookkeeping
	completedMu     sync.Mutex
	completed       []RequestRow
	locationMatches int
	typeMatches     int
	errorCount      int
	fallbackUsed    bool
	uniqueIPs       map[string]struct{}
}

// NewRun validates the config and inserts the audit_runs row.
func NewRun(cfg RunConfig) (*Run, error) {
	if cfg.MaxInFlight < 1 {
		return nil, errors.New("run: max_in_flight must be >= 1")
	}
	if cfg.Lookup == nil || cfg.Store == nil {
		return nil, errors.New("run: lookup and store required")
	}
	r := &Run{cfg: cfg, uniqueIPs: map[string]struct{}{}}
	return r, nil
}

// Subscribe returns an event channel and a cancel func that closes the
// subscription. Multiple subscribers are supported (each gets its own channel).
func (r *Run) Subscribe() (<-chan Event, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan Event, 64)
	if r.finalized {
		close(ch)
		return ch, func() {}
	}
	r.subscribers = append(r.subscribers, ch)
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, c := range r.subscribers {
			if c == ch {
				r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
				break
			}
		}
	}
	return ch, cancel
}

func (r *Run) publish(ev Event) {
	r.mu.Lock()
	subs := append([]chan Event(nil), r.subscribers...)
	r.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
			// drop on slow consumer to keep the run hot
		}
	}
}

func (r *Run) closeSubscribers() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finalized = true
	for _, ch := range r.subscribers {
		close(ch)
	}
	r.subscribers = nil
}

// IsFinalized reports whether the run has reached its terminal state and
// closeSubscribers has run. Used by the SSE handler to choose 204 over a
// stalled stream when a request races a fast-finishing run.
func (r *Run) IsFinalized() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.finalized
}

// Cancel stops dispatching new requests; in-flight requests finish.
func (r *Run) Cancel() {
	r.mu.Lock()
	r.cancelled = true
	cf := r.cancelFn
	r.mu.Unlock()
	if cf != nil {
		cf()
	}
}

// Execute is blocking: it dispatches all requests, finalises, publishes
// finished, and closes subscribers.
func (r *Run) Execute(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	r.mu.Lock()
	r.cancelFn = cancel
	r.mu.Unlock()
	defer cancel()

	sem := semaphore.NewWeighted(int64(r.cfg.MaxInFlight))
	var wg sync.WaitGroup

	for seq := 1; seq <= r.cfg.Spec.RequestCount; seq++ {
		r.mu.Lock()
		if r.cancelled {
			r.mu.Unlock()
			break
		}
		r.mu.Unlock()

		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}

		wg.Add(1)
		go func(seq int) {
			defer sem.Release(1)
			defer wg.Done()

			row := r.runOne(ctx, seq)
			r.recordCompleted(row)
			r.publish(Event{Type: EventRequest, Request: &row})
		}(seq)
	}
	wg.Wait()

	r.finalise()
}

func (r *Run) runOne(ctx context.Context, seq int) RequestRow {
	start := time.Now()
	row := RequestRow{
		RunID: r.cfg.ID, Seq: seq, StartedAt: start.UTC(), GeoSource: "ipapi",
		Attempts: 0,
	}

	var lastRes geo.Result
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		row.Attempts = attempt

		// Per-attempt session rotation when configured.
		reqURL := r.cfg.Spec.ProxyURL
		if r.cfg.Spec.SessionKey != "" {
			rotated, rerr := RotateSession(reqURL, r.cfg.Spec.SessionKey, NewSessionValue())
			if rerr != nil {
				row.DurationMS = int(time.Since(start) / time.Millisecond)
				row.Error = "session rotate: " + rerr.Error()
				row.GeoSource = "none"
				return row
			}
			reqURL = rotated
		}

		proxyURL, perr := url.Parse(reqURL)
		if perr != nil {
			row.DurationMS = int(time.Since(start) / time.Millisecond)
			row.Error = "proxy url parse: " + perr.Error()
			row.GeoSource = "none"
			return row
		}
		hc := &http.Client{
			Timeout: r.cfg.PerRequestTimeout,
			Transport: &http.Transport{
				Proxy:             http.ProxyURL(proxyURL),
				DisableKeepAlives: true,
			},
		}

		rctx, cancel := context.WithTimeout(ctx, r.cfg.PerRequestTimeout)
		res, err := r.cfg.Lookup.Lookup(rctx, hc)
		cancel()

		if err == nil {
			lastRes = res
			lastErr = nil
			break
		}
		lastErr = err

		// No more attempts — exit the loop, the failure is recorded below.
		if attempt == maxAttempts {
			break
		}
		// Honor cancellation while sleeping.
		select {
		case <-time.After(retryBackoff):
		case <-ctx.Done():
			lastErr = ctx.Err()
		}
		if ctx.Err() != nil {
			break
		}
	}

	row.DurationMS = int(time.Since(start) / time.Millisecond)

	if lastErr != nil {
		row.Error = lastErr.Error()
		row.GeoSource = "none"
		return row
	}

	row.ObservedIP = lastRes.IP
	row.ObservedCountry = lastRes.Country
	row.ObservedState = lastRes.State
	row.ObservedCity = lastRes.City
	row.ObservedLat = lastRes.Lat
	row.ObservedLon = lastRes.Lon
	row.IsDatacenter = lastRes.IsDatacenter
	row.IsMobile = lastRes.IsMobile
	row.IsProxy = lastRes.IsProxy
	row.IsVPN = lastRes.IsVPN
	row.ASN = lastRes.ASN
	row.Company = lastRes.Company
	row.GeoSource = lastRes.Source

	v := Evaluate(r.cfg.Expected, lastRes)
	row.LocationMatch = v.LocationMatch
	row.TypeMatch = v.TypeMatch
	return row
}

func (r *Run) recordCompleted(row RequestRow) {
	r.completedMu.Lock()
	r.completed = append(r.completed, row)
	if row.Error == "" {
		if row.LocationMatch {
			r.locationMatches++
		}
		if row.TypeMatch {
			r.typeMatches++
		}
		if row.ObservedIP != "" {
			r.uniqueIPs[row.ObservedIP] = struct{}{}
		}
	} else {
		r.errorCount++
	}
	if row.GeoSource == "maxmind" {
		r.fallbackUsed = true
	}
	r.completedMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), persistTimeout)
	if err := r.cfg.Store.AppendRequest(ctx, row); err != nil {
		// silent for now; integration tests assert via store rows
		_ = err
	}
	cancel()
}

func (r *Run) finalise() {
	r.completedMu.Lock()
	rows := append([]RequestRow(nil), r.completed...)
	uniqueCount := len(r.uniqueIPs)
	r.completedMu.Unlock()

	finished := time.Now().UTC()
	r.mu.Lock()
	cancelled := r.cancelled
	r.mu.Unlock()
	status := StatusCompleted
	if cancelled {
		status = StatusCancelled
	}

	p50, p95 := latencyPercentiles(rows)
	var uic *int
	if uniqueCount > 0 {
		uic = &uniqueCount
	}
	sum := RunSummary{
		ID:                 r.cfg.ID,
		FinishedAt:         &finished,
		Status:             status,
		CompletedCount:     len(rows) - r.errorCount,
		LocationMatchCount: r.locationMatches,
		TypeMatchCount:     r.typeMatches,
		ErrorCount:         r.errorCount,
		LatencyP50MS:       p50,
		LatencyP95MS:       p95,
		FallbackUsed:       r.fallbackUsed,
		UniqueIPCount:      uic,
	}
	// Use a detached context so a cancelled run still gets its row updated.
	ctx, cancel := context.WithTimeout(context.Background(), persistTimeout)
	_ = r.cfg.Store.FinaliseRun(ctx, sum)
	cancel()

	r.publish(Event{Type: EventFinished, Status: status})
	r.closeSubscribers()
}

func latencyPercentiles(rows []RequestRow) (*int, *int) {
	if len(rows) == 0 {
		return nil, nil
	}
	durs := make([]int, 0, len(rows))
	for _, r := range rows {
		if r.Error == "" {
			durs = append(durs, r.DurationMS)
		}
	}
	if len(durs) == 0 {
		return nil, nil
	}
	sort.Ints(durs)
	pick := func(p float64) int {
		idx := int(float64(len(durs)-1) * p)
		return durs[idx]
	}
	p50 := pick(0.50)
	p95 := pick(0.95)
	return &p50, &p95
}
