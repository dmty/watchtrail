package rebuild

import (
	"testing"
	"time"

	"watchtrail/internal/store"
)

func storedSess(id, media string, started time.Time, watched int, completed bool) store.Session {
	return store.Session{
		ID: id, MediaItemID: media, SourceInstance: "i1", SourceKind: "vlc",
		StartedAt: started, EndedAt: started.Add(time.Duration(watched) * time.Second),
		WatchedSeconds: watched, Completed: completed, EventCount: 2,
	}
}

func rebuiltSess(media string, started time.Time, watched int, completed bool) RebuiltSession {
	s := storedSess("", media, started, watched, completed)
	return RebuiltSession{Session: s, EventIDs: []string{"x"}}
}

func TestDiffDetectsChangeAddRemove(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	stored := []store.Session{
		storedSess("a", "m1", base, 60, false),
		storedSess("b", "m2", base.Add(time.Hour), 30, false),
	}
	rebuilt := []RebuiltSession{
		rebuiltSess("m1", base, 90, false),
		rebuiltSess("m3", base.Add(2*time.Hour), 20, false),
	}
	rep := Diff(stored, rebuilt)
	if len(rep.Changed) != 1 || rep.Changed[0].Stored.ID != "a" {
		t.Fatalf("changed = %+v", rep.Changed)
	}
	if len(rep.Added) != 1 || rep.Added[0].Session.MediaItemID != "m3" {
		t.Fatalf("added = %+v", rep.Added)
	}
	if len(rep.Removed) != 1 || rep.Removed[0].ID != "b" {
		t.Fatalf("removed = %+v", rep.Removed)
	}
	if !rep.Drift() {
		t.Fatal("expected drift")
	}
}

func TestDiffNoDrift(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	stored := []store.Session{storedSess("a", "m1", base, 60, true)}
	rebuilt := []RebuiltSession{rebuiltSess("m1", base, 60, true)}
	if Diff(stored, rebuilt).Drift() {
		t.Fatal("expected no drift")
	}
}
