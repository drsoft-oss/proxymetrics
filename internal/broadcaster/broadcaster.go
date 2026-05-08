// Package broadcaster is an in-memory pub/sub for proxy events.
// Subscribers each get a bounded channel; full → drop newest.
package broadcaster

import (
	"sync"
	"sync/atomic"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

const subBufferSize = 256

// Broadcaster fans out events to live subscribers.
type Broadcaster struct {
	mu   sync.RWMutex
	subs map[*Subscription]struct{}
}

func New() *Broadcaster {
	return &Broadcaster{subs: make(map[*Subscription]struct{})}
}

// Publish is non-blocking. Events that don't fit in a subscriber's buffer are
// dropped and counted on that subscriber.
func (b *Broadcaster) Publish(e store.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.subs {
		select {
		case s.ch <- e:
		default:
			s.dropped.Add(1)
		}
	}
}

// SubscriberCount returns the current number of attached subscribers.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

// Subscribe returns a new Subscription. Caller must Close when done.
func (b *Broadcaster) Subscribe() *Subscription {
	s := &Subscription{
		ch: make(chan store.Event, subBufferSize),
		b:  b,
	}
	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()
	return s
}

// Subscription is one attached listener.
type Subscription struct {
	ch        chan store.Event
	dropped   atomic.Int64
	closed    atomic.Bool
	b         *Broadcaster
	closeOnce sync.Once
}

func (s *Subscription) Events() <-chan store.Event { return s.ch }

func (s *Subscription) Dropped() int64 { return s.dropped.Load() }

func (s *Subscription) Close() {
	s.closeOnce.Do(func() {
		s.b.mu.Lock()
		delete(s.b.subs, s)
		s.b.mu.Unlock()
		s.closed.Store(true)
		close(s.ch)
	})
}
