package main

import (
	"net/http/httptest"
	"testing"

	"watchtrail/internal/api"
	"watchtrail/internal/store"
)

func TestProbeAPIUpAndDown(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	srv := httptest.NewServer(api.Handler(repo))
	defer srv.Close()

	if !probeAPI(srv.URL, srv.Client()) {
		t.Fatal("expected API up")
	}
	if probeAPI("http://127.0.0.1:0", srv.Client()) {
		t.Fatal("expected API down")
	}
}
