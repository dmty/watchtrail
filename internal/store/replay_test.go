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
