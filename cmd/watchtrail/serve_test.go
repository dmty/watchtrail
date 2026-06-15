package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"watchtrail/internal/api"
	"watchtrail/internal/ingest"
	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

// TestRootMuxRoutes proves ingest and the read API coexist on one mux.
func TestRootMuxRoutes(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	pipeline := ingest.NewPipeline(repo, sessionize.Config{}, time.Now)

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
