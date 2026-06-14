package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestRenderRecent(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()

	mediaID, err := repo.FindOrCreateMediaItem(ctx, store.MediaItem{
		SourceKind: "vlc", ExternalID: "file:x", IdentityKey: "vlc:file:x",
		Kind: "movie", Title: "Spirited Away",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 14, 21, 0, 0, 0, time.UTC)
	if err := repo.UpsertSession(ctx, store.Session{
		ID: "s1", MediaItemID: mediaID, SourceKind: "vlc", SourceInstance: "laptop",
		StartedAt: now, EndedAt: now.Add(time.Hour), WatchedSeconds: 3725, // 1h2m5s
		Completed: true, EventCount: 5, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	if err := renderRecent(ctx, &sb, repo, 20); err != nil {
		t.Fatalf("renderRecent: %v", err)
	}
	out := sb.String()
	for _, want := range []string{"Spirited Away", "vlc", "1:02:05", "✓"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestFormatWatched(t *testing.T) {
	cases := map[int]string{0: "0:00", 65: "1:05", 3725: "1:02:05"}
	for secs, want := range cases {
		if got := formatWatched(secs); got != want {
			t.Errorf("formatWatched(%d) = %q, want %q", secs, got, want)
		}
	}
}
