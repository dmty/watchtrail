// Package events is a tiny in-process pub/sub for "something changed" pings.
// It carries no domain data: a single contentless signal tells subscribers to
// re-read whatever they display. Publish never blocks the publisher.
package events

import "sync"

// Broker fans a Publish out to every current subscriber. Safe for concurrent use.
type Broker struct {
	mu   sync.Mutex
	subs map[int]chan struct{}
	next int
}

// New returns a ready Broker.
func New() *Broker {
	return &Broker{subs: make(map[int]chan struct{})}
}

// Subscribe registers a subscriber and returns its signal channel plus a cancel
// func that unsubscribes (idempotent). The channel has capacity 1, so repeated
// Publishes with no reader coalesce into a single pending signal.
func (b *Broker) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
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

// Publish sends a non-blocking signal to every subscriber. A subscriber whose
// buffer is full (already has a pending signal) is skipped — it will see the
// pending one. The publisher never blocks on a slow or absent reader.
func (b *Broker) Publish() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
