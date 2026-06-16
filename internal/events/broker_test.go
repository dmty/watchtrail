package events

import (
	"sync"
	"testing"
)

func TestPublishDeliversID(t *testing.T) {
	b := New()
	ch, cancel := b.Subscribe()
	defer cancel()
	b.Publish("m1")
	select {
	case id := <-ch:
		if id != "m1" {
			t.Fatalf("got id %q, want m1", id)
		}
	default:
		t.Fatal("expected a signal after Publish")
	}
}

func TestPublishCoalesces(t *testing.T) {
	b := New()
	ch, cancel := b.Subscribe()
	defer cancel()
	b.Publish("m1")
	b.Publish("m1")
	b.Publish("m1")
	// cap-1 channel: exactly one pending value, extras dropped.
	<-ch
	select {
	case <-ch:
		t.Fatal("expected publishes to coalesce to a single pending signal")
	default:
	}
}

func TestCancelUnsubscribes(t *testing.T) {
	b := New()
	ch, cancel := b.Subscribe()
	cancel()
	cancel() // idempotent
	b.Publish("m1")
	select {
	case <-ch:
		t.Fatal("cancelled subscriber should receive nothing")
	default:
	}
}

func TestConcurrentPublishSubscribe(t *testing.T) {
	// Run with -race.
	b := New()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, cancel := b.Subscribe()
			defer cancel()
			for j := 0; j < 200; j++ {
				b.Publish("m")
				select {
				case <-ch:
				default:
				}
			}
		}()
	}
	wg.Wait()
}
