package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPSingleEventAccepted(t *testing.T) {
	p, repo := newTestPipeline(t)
	srv := httptest.NewServer(p.HTTPHandler("tok"))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ingest", strings.NewReader(rawEvent))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	if n, _ := repo.CountEvents(context.Background()); n != 1 {
		t.Fatalf("CountEvents = %d, want 1", n)
	}
}

func TestHTTPBatchAccepted(t *testing.T) {
	p, repo := newTestPipeline(t)
	srv := httptest.NewServer(p.HTTPHandler("tok"))
	defer srv.Close()

	batch := `[
	  {"v":1,"event_id":"b1","type":"start","occurred_at":"2026-06-14T09:31:02Z","source_kind":"vlc","media":{"external_id":"file:abc"},"position_seconds":0},
	  {"v":1,"event_id":"b2","type":"progress","occurred_at":"2026-06-14T09:31:32Z","source_kind":"vlc","media":{"external_id":"file:abc"},"position_seconds":30}
	]`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ingest", strings.NewReader(batch))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if n, _ := repo.CountEvents(context.Background()); n != 2 {
		t.Fatalf("CountEvents = %d, want 2", n)
	}
}

func TestHTTPRejectsBadToken(t *testing.T) {
	p, _ := newTestPipeline(t)
	srv := httptest.NewServer(p.HTTPHandler("tok"))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ingest", strings.NewReader(rawEvent))
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestHTTPRejectsBadEvent(t *testing.T) {
	p, _ := newTestPipeline(t)
	srv := httptest.NewServer(p.HTTPHandler("tok"))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/ingest", strings.NewReader(`{"v":1,"type":"start"}`))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
