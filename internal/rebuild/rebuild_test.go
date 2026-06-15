package rebuild

import (
	"context"
	"testing"
	"time"

	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

func ev(id, media string, pos float64, at time.Time, typ string) store.Event {
	p := pos
	return store.Event{
		ID: id, MediaItemID: media, SourceKind: "vlc", SourceInstance: "i1",
		Type: typ, PositionSeconds: &p, OccurredAt: at, ReceivedAt: at,
		Raw: []byte(`{}`),
	}
}

var cfg = sessionize.Config{
	SessionGap:          30 * time.Minute,
	CompletionThreshold: 0.9,
	ProgressCadence:     30 * time.Second,
}

func TestReconstructSplitsOnGap(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	events := []store.Event{
		ev("e1", "m1", 0, base, "start"),
		ev("e2", "m1", 30, base.Add(30*time.Second), "progress"),
		ev("e3", "m1", 0, base.Add(2*time.Hour), "start"),
		ev("e4", "m1", 30, base.Add(2*time.Hour+30*time.Second), "progress"),
	}
	got := Reconstruct(events, map[string]*int{"m1": nil}, cfg)
	if len(got) != 2 {
		t.Fatalf("sessions = %d want 2", len(got))
	}
	if len(got[0].EventIDs) != 2 || got[0].EventIDs[0] != "e1" {
		t.Fatalf("session0 events = %v", got[0].EventIDs)
	}
	if got[1].EventIDs[0] != "e3" {
		t.Fatalf("session1 events = %v", got[1].EventIDs)
	}
}

// TestReconstructMatchesLiveAssign is the anti-divergence guarantee: replaying
// the event log in batch yields the same session boundaries and aggregates as
// the live incremental sessionizer (ids excluded — live mints UUIDs).
func TestReconstructMatchesLiveAssign(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	events := []store.Event{
		ev("e1", "m1", 0, base, "start"),
		ev("e2", "m1", 30, base.Add(30*time.Second), "progress"),
		ev("e3", "m1", 60, base.Add(60*time.Second), "progress"),
		ev("e4", "m1", 0, base.Add(3*time.Hour), "start"),
		ev("e5", "m1", 30, base.Add(3*time.Hour+30*time.Second), "progress"),
	}
	if _, err := repo.FindOrCreateMediaItemWithID(ctx, "m1", "T", "vlc"); err != nil {
		t.Fatal(err)
	}
	sz := sessionize.New(repo, cfg, func() time.Time { return base })
	for _, e := range events {
		if err := repo.InsertEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
		if _, err := sz.Assign(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	live, err := repo.AllSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt := Reconstruct(events, map[string]*int{"m1": nil}, cfg)

	if len(live) != len(rebuilt) {
		t.Fatalf("count live=%d rebuilt=%d", len(live), len(rebuilt))
	}
	for i := range live {
		l, b := live[i], rebuilt[i].Session
		if !l.StartedAt.Equal(b.StartedAt) || !l.EndedAt.Equal(b.EndedAt) ||
			l.WatchedSeconds != b.WatchedSeconds || l.Completed != b.Completed ||
			l.EventCount != b.EventCount {
			t.Fatalf("session %d diverged: live=%+v rebuilt=%+v", i, l, b)
		}
	}
}
