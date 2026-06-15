package main

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"watchtrail/internal/api"
	"watchtrail/internal/store"
)

func newClientAPI(t *testing.T) (*apiClient, *store.SQLiteRepo) {
	t.Helper()
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(api.Handler(repo))
	t.Cleanup(func() { srv.Close(); repo.Close() })
	return &apiClient{baseURL: srv.URL, http: srv.Client()}, repo
}

func TestAPIClientSessionsAndDetail(t *testing.T) {
	c, repo := newClientAPI(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo.FindOrCreateMediaItemWithID(ctx, "m1", "Film", "vlc")
	repo.UpsertSession(ctx, store.Session{
		ID: "s1", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
		StartedAt: base, EndedAt: base.Add(time.Minute), WatchedSeconds: 60,
		MaxPositionSeconds: 60, Completed: true, EventCount: 2, CreatedAt: base, UpdatedAt: base,
	})

	rows, err := c.Sessions(ctx, sessionsQuery{Limit: 10})
	if err != nil || len(rows) != 1 || rows[0].Title != "Film" {
		t.Fatalf("sessions = %v err=%v", rows, err)
	}
	detail, ok, err := c.MediaDetail(ctx, "m1")
	if err != nil || !ok || detail.Completions != 1 {
		t.Fatalf("detail = %+v ok=%v err=%v", detail, ok, err)
	}
	if _, ok, _ := c.MediaDetail(ctx, "nope"); ok {
		t.Fatal("expected not found")
	}
	sum, err := c.Summary(ctx, nil, nil)
	if err != nil || sum.Sessions != 1 {
		t.Fatalf("summary = %+v err=%v", sum, err)
	}
}
