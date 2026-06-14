package sessionize

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func newSessionizer(t *testing.T) (*Sessionizer, *store.SQLiteRepo) {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return New(repo, cfg(), func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) }), repo
}

// store an event and run Assign, mimicking the pipeline.
func ingest(t *testing.T, s *Sessionizer, repo *store.SQLiteRepo, mediaID, id, typ string, occ time.Time, pos float64) string {
	t.Helper()
	ctx := context.Background()
	e := store.Event{
		ID: id, MediaItemID: mediaID, SourceKind: "vlc", SourceInstance: "laptop",
		Type: typ, PositionSeconds: &pos, OccurredAt: occ, ReceivedAt: occ, Raw: []byte(`{}`),
	}
	if err := repo.InsertEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	sid, err := s.Assign(ctx, e)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	return sid
}

func TestAssignWithinGapIsOneSession(t *testing.T) {
	s, repo := newSessionizer(t)
	mediaID := seedMediaID(t, repo, "a", 1000)
	t0 := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)

	sid1 := ingest(t, s, repo, mediaID, "e1", "start", t0, 0)
	sid2 := ingest(t, s, repo, mediaID, "e2", "progress", t0.Add(30*time.Second), 30)
	// stop then reopen 5 min later — still one session (gap-only, soft stop)
	ingest(t, s, repo, mediaID, "e3", "stop", t0.Add(40*time.Second), 40)
	sid3 := ingest(t, s, repo, mediaID, "e4", "start", t0.Add(5*time.Minute), 40)

	if sid1 != sid2 || sid2 != sid3 {
		t.Fatalf("expected one session, got %s %s %s", sid1, sid2, sid3)
	}
	got, found, _ := repo.LatestSessionFor(context.Background(), mediaID, "laptop")
	// e1 start + e2 progress + e3 stop + e4 reopen = 4 events, one session
	if !found || got.EventCount != 4 {
		t.Fatalf("session = %+v", got)
	}
}

func TestAssignPastGapIsNewSession(t *testing.T) {
	s, repo := newSessionizer(t)
	mediaID := seedMediaID(t, repo, "a", 1000)
	t0 := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)

	sid1 := ingest(t, s, repo, mediaID, "e1", "start", t0, 0)
	// 31 minutes later: beyond the 30-min gap -> new session
	sid2 := ingest(t, s, repo, mediaID, "e2", "start", t0.Add(31*time.Minute), 0)

	if sid1 == sid2 {
		t.Fatalf("expected two sessions, both = %s", sid1)
	}
}

func TestAssignBackfillsEventSessionID(t *testing.T) {
	s, repo := newSessionizer(t)
	mediaID := seedMediaID(t, repo, "a", 1000)
	t0 := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)
	sid := ingest(t, s, repo, mediaID, "e1", "start", t0, 0)

	var got string
	if err := repo.DB().QueryRowContext(context.Background(),
		`SELECT session_id FROM watch_event WHERE id = 'e1'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != sid {
		t.Fatalf("event session_id = %q, want %q", got, sid)
	}
}

func seedMediaID(t *testing.T, repo *store.SQLiteRepo, identity string, duration int) string {
	t.Helper()
	id, err := repo.FindOrCreateMediaItem(context.Background(), store.MediaItem{
		SourceKind: "vlc", ExternalID: identity, IdentityKey: "vlc:" + identity,
		Kind: "movie", Title: "M", DurationSeconds: &duration,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
