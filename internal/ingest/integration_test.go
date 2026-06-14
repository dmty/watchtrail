package ingest

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEndToEndBothTransportsIdempotent verifies that the same event delivered
// over HTTP and then TCP is stored exactly once (cross-transport idempotency).
func TestEndToEndBothTransportsIdempotent(t *testing.T) {
	p, repo := newTestPipeline(t)
	ctx := context.Background()

	// HTTP delivery.
	srv := httptest.NewServer(p.HTTPHandler("tok"))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ingest", strings.NewReader(rawEvent))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("HTTP status = %d, want 202", resp.StatusCode)
	}
	_ = resp.Body.Close()

	if n, _ := repo.CountEvents(ctx); n != 1 {
		t.Fatalf("after HTTP: CountEvents = %d, want 1", n)
	}

	// TCP delivery of the SAME event_id — must dedupe, not double-count.
	addr := startTCP(t, p, "tok")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = conn.Write([]byte("tok\n" + compact(rawEvent) + "\n"))
	_ = conn.Close()

	waitForCount(t, repo, 1) // still exactly one
}
