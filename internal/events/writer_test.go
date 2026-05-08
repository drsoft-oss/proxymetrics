package events_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/drsoft-oss/proxymetrics/internal/events"
	"github.com/drsoft-oss/proxymetrics/internal/store"
)

type captureStore struct {
	mu      sync.Mutex
	written [][]store.Event
	delay   time.Duration
}

func (c *captureStore) WriteEvents(ctx context.Context, batch []store.Event) error {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]store.Event, len(batch))
	copy(cp, batch)
	c.written = append(c.written, cp)
	return nil
}

func TestWriter_FlushesOnBatchSize(t *testing.T) {
	cs := &captureStore{}
	w := events.NewWriter(events.Config{Store: cs, ChannelSize: 10, BatchSize: 3, FlushInterval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	for i := 0; i < 3; i++ {
		w.Submit(store.Event{RequestID: "x", Vendor: "v", Type: "residential", StatusClass: "2xx"})
	}
	// Allow one flush.
	time.Sleep(50 * time.Millisecond)

	cs.mu.Lock()
	if len(cs.written) != 1 || len(cs.written[0]) != 3 {
		t.Fatalf("written: %#v", cs.written)
	}
	cs.mu.Unlock()

	cancel()
	<-done
}

func TestWriter_FlushesOnInterval(t *testing.T) {
	cs := &captureStore{}
	w := events.NewWriter(events.Config{Store: cs, ChannelSize: 10, BatchSize: 100, FlushInterval: 30 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	w.Submit(store.Event{RequestID: "y", Vendor: "v", Type: "residential", StatusClass: "2xx"})
	time.Sleep(80 * time.Millisecond)

	cs.mu.Lock()
	defer cs.mu.Unlock()
	if len(cs.written) == 0 {
		t.Fatal("no flush after interval")
	}
}

func TestWriter_DropOnFull(t *testing.T) {
	cs := &captureStore{delay: 50 * time.Millisecond}
	w := events.NewWriter(events.Config{Store: cs, ChannelSize: 1, BatchSize: 1, FlushInterval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	var dropped atomic.Int64
	for i := 0; i < 100; i++ {
		if !w.Submit(store.Event{RequestID: "z", Vendor: "v", Type: "residential", StatusClass: "2xx"}) {
			dropped.Add(1)
		}
	}
	if dropped.Load() == 0 {
		t.Fatal("expected at least some drops under blocked writer")
	}
	if w.DroppedCount() != dropped.Load() {
		t.Fatalf("counter mismatch: %d vs %d", w.DroppedCount(), dropped.Load())
	}
}

type captureObserver struct {
	mu   sync.Mutex
	seen []string // "vendor|type" pairs in call order
}

func (c *captureObserver) Observe(ctx context.Context, vendor, typ string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = append(c.seen, vendor+"|"+typ)
	return nil
}

func (c *captureObserver) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.seen))
	copy(out, c.seen)
	return out
}

func TestWriter_ObservesEachVendorTypeOncePerFlush(t *testing.T) {
	cs := &captureStore{}
	obs := &captureObserver{}
	w := events.NewWriter(events.Config{
		Store:           cs,
		ChannelSize:     10,
		BatchSize:       4,
		FlushInterval:   time.Hour,
		ProfileObserver: obs,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	// Two distinct (vendor,type) combos, multiple events each.
	w.Submit(store.Event{RequestID: "1", Vendor: "brightdata", Type: "residential", StatusClass: "2xx"})
	w.Submit(store.Event{RequestID: "2", Vendor: "brightdata", Type: "residential", StatusClass: "2xx"})
	w.Submit(store.Event{RequestID: "3", Vendor: "oxylabs", Type: "datacenter", StatusClass: "2xx"})
	w.Submit(store.Event{RequestID: "4", Vendor: "oxylabs", Type: "datacenter", StatusClass: "2xx"})

	// Wait for flush + observer calls.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(obs.snapshot()) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	got := obs.snapshot()
	if len(got) != 2 {
		t.Fatalf("expected 2 observer calls (one per distinct vendor|type), got %d: %v", len(got), got)
	}
	want := map[string]bool{"brightdata|residential": false, "oxylabs|datacenter": false}
	for _, s := range got {
		if _, ok := want[s]; !ok {
			t.Errorf("unexpected observer call: %q", s)
			continue
		}
		want[s] = true
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("expected observer call for %q, missing", k)
		}
	}

	cancel()
	<-done
}

func TestWriter_NoObserverConfigDoesNotPanic(t *testing.T) {
	cs := &captureStore{}
	// ProfileObserver intentionally nil — must not panic.
	w := events.NewWriter(events.Config{
		Store:         cs,
		ChannelSize:   10,
		BatchSize:     1,
		FlushInterval: time.Hour,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	w.Submit(store.Event{RequestID: "x", Vendor: "v", Type: "residential", StatusClass: "2xx"})
	time.Sleep(50 * time.Millisecond)
}

func TestWriter_DrainsOnContextCancel(t *testing.T) {
	cs := &captureStore{}
	w := events.NewWriter(events.Config{Store: cs, ChannelSize: 100, BatchSize: 100, FlushInterval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	for i := 0; i < 50; i++ {
		w.Submit(store.Event{RequestID: "a", Vendor: "v", Type: "residential", StatusClass: "2xx"})
	}
	cancel()
	<-done

	cs.mu.Lock()
	defer cs.mu.Unlock()
	total := 0
	for _, b := range cs.written {
		total += len(b)
	}
	if total != 50 {
		t.Fatalf("drain: wrote %d, want 50", total)
	}
}
