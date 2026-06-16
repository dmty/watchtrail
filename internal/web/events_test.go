// internal/web/events_test.go
package web

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"watchtrail/internal/events"
)

func TestEventsStreamPushesUpdate(t *testing.T) {
	broker := events.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", handleEvents(broker))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}

	sc := bufio.NewScanner(resp.Body)
	readLine := func() string {
		if sc.Scan() {
			return sc.Text()
		}
		return "<eof>"
	}
	// The handler subscribes before writing the connected comment, so a Publish
	// after we read it is guaranteed to be delivered.
	if l := readLine(); l != ": connected" {
		t.Fatalf("first line = %q", l)
	}
	broker.Publish("media-xyz")
	sawEvent, sawData := false, false
	for i := 0; i < 8; i++ {
		switch readLine() {
		case "event: update":
			sawEvent = true
		case "data: media-xyz":
			sawData = true
		}
		if sawEvent && sawData {
			return // success
		}
	}
	t.Fatal("did not receive 'event: update' with 'data: media-xyz'")
}

// nonFlusherWriter wraps a ResponseRecorder but deliberately does NOT promote
// its Flush method, so it does not satisfy http.Flusher. (As of Go 1.26
// httptest.ResponseRecorder itself implements http.Flusher, so it cannot be used
// to exercise the guard directly.)
type nonFlusherWriter struct {
	rec *httptest.ResponseRecorder
}

func (w *nonFlusherWriter) Header() http.Header         { return w.rec.Header() }
func (w *nonFlusherWriter) Write(b []byte) (int, error) { return w.rec.Write(b) }
func (w *nonFlusherWriter) WriteHeader(code int)        { w.rec.WriteHeader(code) }

func TestEventsRequiresFlusher(t *testing.T) {
	// A ResponseWriter that is not a Flusher → 500.
	rec := httptest.NewRecorder()
	w := &nonFlusherWriter{rec: rec}
	req := httptest.NewRequest("GET", "/events", nil)
	handleEvents(events.New())(w, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
