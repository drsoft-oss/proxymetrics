package broadcaster_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anonymous-proxies/proxymetrics/internal/broadcaster"
	"github.com/anonymous-proxies/proxymetrics/internal/store"
)

func TestPublish_NoSubscribers_NoOp(t *testing.T) {
	b := broadcaster.New()
	b.Publish(store.Event{RequestID: "x"}) // must not panic / block
}

func TestSubscribe_ReceivesPublishedEvents(t *testing.T) {
	b := broadcaster.New()
	sub := b.Subscribe()
	defer sub.Close()

	go b.Publish(store.Event{RequestID: "r1"})

	select {
	case e := <-sub.Events():
		if e.RequestID != "r1" {
			t.Fatalf("got %q", e.RequestID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestClose_StopsDelivery(t *testing.T) {
	b := broadcaster.New()
	sub := b.Subscribe()
	sub.Close()
	if b.SubscriberCount() != 0 {
		t.Fatalf("count after close: %d", b.SubscriberCount())
	}
	b.Publish(store.Event{RequestID: "r"})
}

func TestPublish_DropsWhenSubscriberFull(t *testing.T) {
	b := broadcaster.New()
	sub := b.Subscribe()
	defer sub.Close()

	for i := 0; i < 1000; i++ {
		b.Publish(store.Event{RequestID: "r"})
	}
	if sub.Dropped() == 0 {
		t.Fatal("expected drops")
	}
}

func TestPublish_ConcurrentSafe(t *testing.T) {
	b := broadcaster.New()
	const subs = 8
	var wg sync.WaitGroup
	var received atomic.Int64

	for i := 0; i < subs; i++ {
		s := b.Subscribe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range s.Events() {
				received.Add(1)
				if received.Load() >= int64(subs*100) {
					s.Close()
					return
				}
			}
		}()
	}

	for i := 0; i < 100; i++ {
		b.Publish(store.Event{RequestID: "r"})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	if received.Load() == 0 {
		t.Fatal("no events received")
	}
}
