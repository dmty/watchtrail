package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"watchtrail/internal/api"
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
