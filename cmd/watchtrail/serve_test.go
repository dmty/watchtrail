package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"watchtrail/internal/api"
	"watchtrail/internal/tlsca"
	"watchtrail/internal/events"
	"watchtrail/internal/ingest"
	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
	"watchtrail/internal/thumb"
	"watchtrail/internal/web"
)

// TestRootMuxRoutes proves ingest and the read API coexist on one mux.
func TestRootMuxRoutes(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	pipeline := ingest.NewPipeline(repo, sessionize.Config{}, time.Now, nil)

	root := http.NewServeMux()
	root.Handle("/ingest", pipeline.HTTPHandler(""))
	root.Handle("/api/v1/", api.Handler(repo))

	srv := httptest.NewServer(root)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health status %d", resp.StatusCode)
	}

	resp2, err := http.Get(srv.URL + "/ingest")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("ingest GET status %d want 405", resp2.StatusCode)
	}
}

func TestRootMuxServesDashboard(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	webHandler, err := web.Handler(repo, events.New(), thumb.Build(t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	root := http.NewServeMux()
	root.Handle("/api/v1/", api.Handler(repo))
	root.Handle("/", webHandler)

	srv := httptest.NewServer(root)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("dashboard / status %d", resp.StatusCode)
	}
	resp2, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("api health status %d", resp2.StatusCode)
	}
}

func TestRootMuxServesEvents(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	broker := events.New()
	webHandler, err := web.Handler(repo, broker, thumb.Build(t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	root := http.NewServeMux()
	root.Handle("/", webHandler)
	srv := httptest.NewServer(root)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("/events status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("/events content-type %q", ct)
	}
	cancel()
}

func TestHTTPSRedirect(t *testing.T) {
	h := httpsRedirect("8443")
	req := httptest.NewRequest(http.MethodGet, "http://watchtrail.local:8765/some/path?x=1", nil)
	req.Host = "watchtrail.local:8765"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want 308", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://") {
		t.Fatalf("Location %q does not start with https://", loc)
	}
	if !strings.Contains(loc, ":8443") {
		t.Fatalf("Location %q does not contain port 8443", loc)
	}
	if !strings.Contains(loc, "/some/path") {
		t.Fatalf("Location %q does not preserve path", loc)
	}
	if !strings.Contains(loc, "x=1") {
		t.Fatalf("Location %q does not preserve query", loc)
	}
}

func TestCACertHandler404WhenAbsent(t *testing.T) {
	h := caCertHandler(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "http://watchtrail.local/ca.crt", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestCACertHandlerServesCA(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := tlsca.Enable(dir, []string{"127.0.0.1"}, time.Now()); err != nil {
		t.Fatalf("enable: %v", err)
	}
	h := caCertHandler(dir)
	req := httptest.NewRequest(http.MethodGet, "http://watchtrail.local/ca.crt", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-x509-ca-cert" {
		t.Fatalf("Content-Type = %q, want application/x-x509-ca-cert", ct)
	}
}
