package store

import (
	"context"
	"testing"
	"time"
)

func seedMedia(t *testing.T, repo *SQLiteRepo, identity string, duration *int) string {
	t.Helper()
	id, err := repo.FindOrCreateMediaItem(context.Background(), MediaItem{
		SourceKind: "vlc", ExternalID: identity, IdentityKey: "vlc:" + identity,
		Kind: "movie", Title: "Movie " + identity, DurationSeconds: duration,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMediaDuration(t *testing.T) {
	repo := openTemp(t)
	ctx := context.Background()
	withDur := seedMedia(t, repo, "a", func() *int { v := 7500; return &v }())
	noDur := seedMedia(t, repo, "b", nil)

	d, err := repo.MediaDuration(ctx, withDur)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || *d != 7500 {
		t.Errorf("MediaDuration(withDur) = %v, want 7500", d)
	}
	d, err = repo.MediaDuration(ctx, noDur)
	if err != nil {
		t.Fatal(err)
	}
	if d != nil {
		t.Errorf("MediaDuration(noDur) = %v, want nil", d)
	}
}

func TestUpsertAndLatestSession(t *testing.T) {
	repo := openTemp(t)
	ctx := context.Background()
	mediaID := seedMedia(t, repo, "a", nil)
	now := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)

	s := Session{
		ID: "sess-1", MediaItemID: mediaID, SourceKind: "vlc", SourceInstance: "laptop",
		StartedAt: now, EndedAt: now.Add(time.Minute), WatchedSeconds: 60,
		MaxPositionSeconds: 60, Completed: false, EventCount: 3,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.UpsertSession(ctx, s); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, found, err := repo.LatestSessionFor(ctx, mediaID, "laptop")
	if err != nil || !found {
		t.Fatalf("LatestSessionFor: found=%v err=%v", found, err)
	}
	if got.ID != "sess-1" || got.WatchedSeconds != 60 || got.EventCount != 3 {
		t.Errorf("got %+v", got)
	}

	// Update via upsert: created_at must be preserved, fields updated.
	s.WatchedSeconds = 120
	s.Completed = true
	s.EventCount = 5
	s.UpdatedAt = now.Add(time.Hour)
	s.CreatedAt = now.Add(time.Hour) // should be ignored on conflict
	if err := repo.UpsertSession(ctx, s); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _, _ = repo.LatestSessionFor(ctx, mediaID, "laptop")
	if got.WatchedSeconds != 120 || !got.Completed || got.EventCount != 5 {
		t.Errorf("after update: %+v", got)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want preserved %v", got.CreatedAt, now)
	}
}

func TestLatestSessionForMissing(t *testing.T) {
	repo := openTemp(t)
	_, found, err := repo.LatestSessionFor(context.Background(), "nope", "x")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("found = true, want false")
	}
}

func TestRecentSessions(t *testing.T) {
	repo := openTemp(t)
	ctx := context.Background()
	mediaID := seedMedia(t, repo, "a", nil)
	now := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)

	older := Session{
		ID: "old", MediaItemID: mediaID, SourceKind: "vlc", SourceInstance: "laptop",
		StartedAt: now, EndedAt: now, WatchedSeconds: 10, CreatedAt: now, UpdatedAt: now,
	}
	newer := Session{
		ID: "new", MediaItemID: mediaID, SourceKind: "vlc", SourceInstance: "laptop",
		StartedAt: now.Add(time.Hour), EndedAt: now.Add(time.Hour), WatchedSeconds: 20,
		Completed: true, CreatedAt: now, UpdatedAt: now,
	}
	for _, s := range []Session{older, newer} {
		if err := repo.UpsertSession(ctx, s); err != nil {
			t.Fatal(err)
		}
	}

	views, err := repo.RecentSessions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("len = %d, want 2", len(views))
	}
	if views[0].WatchedSeconds != 20 || !views[0].Completed { // newest first
		t.Errorf("views[0] = %+v, want newest", views[0])
	}
	if views[0].Title != "Movie a" || views[0].SourceKind != "vlc" {
		t.Errorf("join wrong: %+v", views[0])
	}

	// limit is honored
	views, err = repo.RecentSessions(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 {
		t.Fatalf("limited len = %d, want 1", len(views))
	}
}

func TestSetEventSessionAndEventsForSession(t *testing.T) {
	repo := openTemp(t)
	ctx := context.Background()
	mediaID := seedMedia(t, repo, "a", nil)
	occ := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)

	for i, id := range []string{"e2", "e1"} { // insert out of occurred_at order
		pos := float64(i * 30)
		if err := repo.InsertEvent(ctx, Event{
			ID: id, MediaItemID: mediaID, SourceKind: "vlc", Type: "progress",
			PositionSeconds: &pos, OccurredAt: occ.Add(time.Duration(30*(1-i)) * time.Second),
			ReceivedAt: occ, Raw: []byte(`{}`),
		}); err != nil {
			t.Fatal(err)
		}
		if err := repo.SetEventSession(ctx, id, "sess-1"); err != nil {
			t.Fatal(err)
		}
	}

	evs, err := repo.EventsForSession(ctx, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("len = %d, want 2", len(evs))
	}
	if !evs[0].OccurredAt.Before(evs[1].OccurredAt) {
		t.Errorf("events not ordered by occurred_at: %v, %v", evs[0].OccurredAt, evs[1].OccurredAt)
	}
	if evs[0].Type != "progress" || evs[0].MediaItemID != mediaID {
		t.Errorf("scan wrong: %+v", evs[0])
	}
}
