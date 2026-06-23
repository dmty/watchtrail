package discovery

import (
	"context"
	"testing"
	"time"
)

func TestRegisterShutdownOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	svc, err := Register(ctx, "watchtrail-test", 8765)
	if err != nil {
		t.Skipf("mdns not available in this env: %v", err)
	}
	cancel()
	// Give the goroutine a moment to drop the registration.
	deadline := time.Now().Add(2 * time.Second)
	for !svc.IsClosed() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !svc.IsClosed() {
		t.Errorf("service did not close after ctx cancel")
	}
}

func TestExplicitShutdownIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc, err := Register(ctx, "watchtrail-test-2", 8765)
	if err != nil {
		t.Skipf("mdns not available in this env: %v", err)
	}
	svc.Shutdown()
	svc.Shutdown() // must not panic / double-close
	if !svc.IsClosed() {
		t.Errorf("expected closed after explicit shutdown")
	}
}
