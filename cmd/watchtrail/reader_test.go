package main

import (
	"context"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestStoreReader(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	if _, err := repo.FindOrCreateMediaItemWithID(ctx, "m1", "Film", "vlc"); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertSession(ctx, store.Session{
		ID: "s1", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
		StartedAt: base, EndedAt: base.Add(time.Minute), WatchedSeconds: 60,
		MaxPositionSeconds: 60, Completed: true, EventCount: 2, CreatedAt: base, UpdatedAt: base,
	}); err != nil {
		t.Fatal(err)
	}

	var rd reader = &storeReader{repo: repo}
	rows, err := rd.Sessions(ctx, sessionsQuery{Limit: 10})
	if err != nil || len(rows) != 1 || rows[0].ID != "s1" {
		t.Fatalf("sessions = %v err=%v", rows, err)
	}
	detail, ok, err := rd.MediaDetail(ctx, "m1")
	if err != nil || !ok {
		t.Fatalf("detail ok=%v err=%v", ok, err)
	}
	if detail.Title != "Film" || detail.Starts != 1 || detail.Completions != 1 {
		t.Fatalf("detail = %+v", detail)
	}
	sum, err := rd.Summary(ctx, nil, nil)
	if err != nil || sum.Sessions != 1 {
		t.Fatalf("summary = %+v err=%v", sum, err)
	}
}
