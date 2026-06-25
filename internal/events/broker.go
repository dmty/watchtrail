// Package events is a tiny in-process pub/sub for "something changed" pings.
// The signal is a media item id, telling subscribers which item changed so they
// can decide whether to refresh. Publish never blocks the publisher.
package events

import "sync"

// Broker fans a Publish out to every current subscriber. Safe for concurrent use.
type Broker struct {
	mu   sync.Mutex
	subs map[int]chan string
	next int
	done chan struct{}
}

// New returns a ready Broker.
func New() *Broker {
	return &Broker{subs: make(map[int]chan string), done: make(chan struct{})}
}

// Done returns a channel closed by Close. Long-lived subscribers select on it
// to unblock during server shutdown.
func (b *Broker) Done() <-chan struct{} { return b.done }

// Close signals subscribers to exit. Idempotent.
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-b.done:
	default:
		close(b.done)
	}
}

// Subscribe registers a subscriber and returns its signal channel (carrying the
// changed media id) plus a cancel func that unsubscribes (idempotent). The channel
// has capacity 1, so repeated Publishes with no reader coalesce to a single pending
// signal.
func (b *Broker) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 1)
	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.subs, id)
		b.mu.Unlock()
	}
	return ch, cancel
}

// Publish sends mediaID to every subscriber, non-blocking. A subscriber whose
// buffer is full (already has a pending signal) is skipped. The publisher never
// blocks on a slow or absent reader.
func (b *Broker) Publish(mediaID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		select {
		case ch <- mediaID:
		default:
		}
	}
}
