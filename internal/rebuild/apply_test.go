package rebuild

import (
	"context"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestApplyWritesAndIsIdempotent(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	if _, err := repo.FindOrCreateMediaItemWithID(ctx, "m1", "T", "vlc"); err != nil {
		t.Fatal(err)
	}
	for _, e := range []store.Event{
		ev("e1", "m1", 0, base, "start"),
		ev("e2", "m1", 30, base.Add(30*time.Second), "progress"),
	} {
		if err := repo.InsertEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	events, _ := repo.AllEvents(ctx)
	durs, _ := repo.AllMediaDurations(ctx)
	rebuilt := Reconstruct(events, durs, cfg)

	clock := func() time.Time { return base }
	if err := Apply(ctx, repo, rebuilt, clock); err != nil {
		t.Fatal(err)
	}
	stored, _ := repo.AllSessions(ctx)
	if Diff(stored, Reconstruct(events, durs, cfg)).Drift() {
		t.Fatal("drift after apply")
	}
	if stored[0].ID == "" {
		t.Fatal("apply must mint a session id")
	}
}
