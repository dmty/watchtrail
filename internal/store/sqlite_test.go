package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func openTemp(t *testing.T) *SQLiteRepo {
	t.Helper()
	repo, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestMigrationsCreateTables(t *testing.T) {
	repo := openTemp(t)
	for _, table := range []string{"media_item", "watch_event", "watch_session", "schema_migrations"} {
		var name string
		err := repo.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	r1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = r1.Close()
	r2, err := Open(path) // re-running migrate must not error
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	_ = r2.Close()
}

func sampleMedia() MediaItem {
	return MediaItem{
		SourceKind:  "vlc",
		ExternalID:  "file:abc",
		IdentityKey: "vlc:file:abc",
		Kind:        "movie",
		Title:       "Spirited Away",
	}
}

func sampleEvent(id, mediaID string) Event {
	pos := 1342.0
	now := time.Date(2026, 6, 14, 9, 31, 2, 0, time.UTC)
	return Event{
		ID:              id,
		MediaItemID:     mediaID,
		SourceKind:      "vlc",
		SourceInstance:  "laptop-vlc",
		Type:            "progress",
		PositionSeconds: &pos,
		OccurredAt:      now,
		ReceivedAt:      now,
		Raw:             []byte(`{"v":1}`),
	}
}

func TestFindOrCreateMediaItemDedup(t *testing.T) {
	repo := openTemp(t)
	ctx := context.Background()

	id1, err := repo.FindOrCreateMediaItem(ctx, sampleMedia())
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	id2, err := repo.FindOrCreateMediaItem(ctx, sampleMedia())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("identity_key not deduped: %q != %q", id1, id2)
	}
}

func TestInsertEventIsIdempotent(t *testing.T) {
	repo := openTemp(t)
	ctx := context.Background()
	mediaID, err := repo.FindOrCreateMediaItem(ctx, sampleMedia())
	if err != nil {
		t.Fatal(err)
	}
	ev := sampleEvent("event-1", mediaID)
	if err := repo.InsertEvent(ctx, ev); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := repo.InsertEvent(ctx, ev); err != nil {
		t.Fatalf("re-insert should be a no-op, got: %v", err)
	}
	n, err := repo.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("CountEvents = %d, want 1 (idempotent)", n)
	}
}
