// Package events provides a buffered event writer that batches into a Store.
package events

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

// Sink is the slice of store.Store actually used by the writer.
type Sink interface {
	WriteEvents(ctx context.Context, batch []store.Event) error
}

// ProfileObserver is notified once per distinct (vendor, type) pair seen in a
// flushed batch. The writer is the auto-discovery path that turns observed
// proxy traffic into registry rows.
type ProfileObserver interface {
	Observe(ctx context.Context, vendor, typ string) error
}

type Config struct {
	Store         Sink
	ChannelSize   int
	BatchSize     int
	FlushInterval time.Duration
	// ShutdownSignal, when non-nil, is closed by the orchestrator after the
	// proxy server has finished draining in-flight requests. The writer waits
	// on this before beginning its drain. If nil, ctx.Done() triggers drain
	// (legacy behaviour, used by tests).
	ShutdownSignal <-chan struct{}
	// ProfileObserver, when non-nil, is called once per distinct (vendor, type)
	// pair after each successful flush. Errors are swallowed — auto-discovery
	// is best-effort and must never block the writer.
	ProfileObserver ProfileObserver
}

// Writer is the event-writer goroutine handle.
type Writer struct {
	cfg     Config
	ch      chan store.Event
	dropped atomic.Int64
}

func NewWriter(cfg Config) *Writer {
	return &Writer{cfg: cfg, ch: make(chan store.Event, cfg.ChannelSize)}
}

// Submit returns false if the channel was full and the event was dropped.
// Hot-path callers must check the return and increment their own metric if desired.
func (w *Writer) Submit(e store.Event) bool {
	select {
	case w.ch <- e:
		return true
	default:
		w.dropped.Add(1)
		return false
	}
}

// DroppedCount returns the cumulative count of events dropped due to a full channel.
func (w *Writer) DroppedCount() int64 { return w.dropped.Load() }

// observe notifies the configured ProfileObserver once per distinct
// (vendor, type) pair in the batch. It runs synchronously inside the writer
// goroutine so callers don't have to coordinate goroutine lifetime; the
// writer is already off the proxy hot path. Observer errors are swallowed.
func (w *Writer) observe(ctx context.Context, batch []store.Event) {
	if w.cfg.ProfileObserver == nil {
		return
	}
	type key struct{ vendor, typ string }
	seen := make(map[key]struct{}, len(batch))
	for _, e := range batch {
		k := key{e.Vendor, e.Type}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		_ = w.cfg.ProfileObserver.Observe(ctx, k.vendor, k.typ)
	}
}

// Run consumes the channel until ctx is canceled (or ShutdownSignal is closed),
// then drains and flushes. When ShutdownSignal is set, the writer waits for
// it before draining so that in-flight proxy requests can still submit events
// during the proxy server's graceful shutdown window.
func (w *Writer) Run(ctx context.Context) error {
	tick := time.NewTicker(w.cfg.FlushInterval)
	defer tick.Stop()

	batch := make([]store.Event, 0, w.cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Use a fresh background context for the actual write so a canceled parent
		// does not abort the final drain. WriteEvents implementations must respect
		// any internal timeouts.
		writeCtx := context.Background()
		_ = w.cfg.Store.WriteEvents(writeCtx, batch)
		w.observe(writeCtx, batch)
		batch = batch[:0]
	}

	// drainSignal fires when it is safe to begin the final drain.
	drainSignal := make(chan struct{})
	go func() {
		if w.cfg.ShutdownSignal != nil {
			// Wait for the orchestrator to confirm the proxy server has drained,
			// or fall through if ctx is canceled without the signal being closed.
			select {
			case <-w.cfg.ShutdownSignal:
			case <-ctx.Done():
			}
		} else {
			<-ctx.Done()
		}
		close(drainSignal)
	}()

	for {
		select {
		case <-drainSignal:
			// Drain whatever is still queued.
			for {
				select {
				case e := <-w.ch:
					batch = append(batch, e)
					if len(batch) >= w.cfg.BatchSize {
						flush()
					}
				default:
					flush()
					return nil
				}
			}
		case e := <-w.ch:
			batch = append(batch, e)
			if len(batch) >= w.cfg.BatchSize {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}
