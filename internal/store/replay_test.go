package store

import (
	"context"
	"testing"
	"time"
)

func seedEvent(t *testing.T, r *SQLiteRepo, id, mediaID, source, typ string, pos float64, at time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := r.FindOrCreateMediaItemWithID(ctx, mediaID, "T", source); err != nil {
		t.Fatal(err)
	}
	p := pos
	if err := r.InsertEvent(ctx, Event{
		ID: id, MediaItemID: mediaID, SourceKind: source, SourceInstance: "i1",
		Type: typ, PositionSeconds: &p, OccurredAt: at, ReceivedAt: at, Raw: []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAllEventsAndDurations(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	seedEvent(t, r, "e1", "m1", "vlc", "start", 0, base)
	seedEvent(t, r, "e2", "m1", "vlc", "progress", 30, base.Add(30*time.Second))

	evs, err := r.AllEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0].ID != "e1" {
		t.Fatalf("events = %d", len(evs))
	}
	durs, err := r.AllMediaDurations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := durs["m1"]; !ok {
		t.Fatal("missing m1 duration key")
	}
}

func TestAllEventsExcludesDeletedMedia(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	seedEvent(t, r, "e1", "m1", "vlc", "start", 0, base)
	seedEvent(t, r, "e2", "m2", "vlc", "start", 0, base.Add(time.Second))

	if _, err := r.SoftDeleteMedia(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	evs, err := r.AllEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].MediaItemID != "m2" {
		t.Fatalf("AllEvents = %+v; want only m2's event", evs)
	}
}

func TestReplaceAllSessions(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	seedEvent(t, r, "e1", "m1", "vlc", "start", 0, base)
	seedEvent(t, r, "e2", "m1", "vlc", "progress", 30, base.Add(30*time.Second))

	writes := []SessionWrite{{
		Session: Session{
			ID: "new-sess", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
			StartedAt: base, EndedAt: base.Add(30 * time.Second), WatchedSeconds: 30,
			MaxPositionSeconds: 30, Completed: false, EventCount: 2,
			CreatedAt: base, UpdatedAt: base,
		},
		EventIDs: []string{"e1", "e2"},
	}}
	if err := r.ReplaceAllSessions(ctx, writes); err != nil {
		t.Fatal(err)
	}
	sessions, err := r.AllSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "new-sess" {
		t.Fatalf("sessions = %+v", sessions)
	}
	evs, err := r.EventsForSession(ctx, "new-sess")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("event repoint = %d want 2", len(evs))
	}

	if err := r.ReplaceAllSessions(ctx, writes); err != nil {
		t.Fatal(err)
	}
	again, _ := r.AllSessions(ctx)
	if len(again) != 1 {
		t.Fatalf("after re-replace = %d want 1", len(again))
	}
}
