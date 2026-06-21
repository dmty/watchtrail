package main

import (
	"bytes"
	"context"
	"testing"
	"time"

	"watchtrail/internal/rebuild"
	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

func TestRebuildReportNoWrite(t *testing.T) {
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
	p := 30.0
	if err := repo.InsertEvent(ctx, store.Event{
		ID: "e1", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
		Type: "start", PositionSeconds: &p, OccurredAt: base, ReceivedAt: base, Raw: []byte("{}"),
	}); err != nil {
		t.Fatal(err)
	}
	cfg := sessionize.Config{SessionGap: 30 * time.Minute, CompletionThreshold: 0.9, ProgressCadence: 30 * time.Second}

	var out bytes.Buffer
	drift, err := runRebuildReport(ctx, &out, repo, cfg, false, func() time.Time { return base })
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift (no sessions stored yet, one rebuilt)")
	}
	if out.Len() == 0 {
		t.Fatal("expected a printed report")
	}
	sessions, _ := repo.AllSessions(ctx)
	if len(sessions) != 0 {
		t.Fatalf("verify must not write; got %d sessions", len(sessions))
	}
}

func TestRebuildWriteResolvesDrift(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo.FindOrCreateMediaItemWithID(ctx, "m1", "T", "vlc")
	p := 30.0
	repo.InsertEvent(ctx, store.Event{
		ID: "e1", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
		Type: "start", PositionSeconds: &p, OccurredAt: base, ReceivedAt: base, Raw: []byte("{}"),
	})
	cfg := sessionize.Config{SessionGap: 30 * time.Minute, CompletionThreshold: 0.9, ProgressCadence: 30 * time.Second}

	var out bytes.Buffer
	if _, err := runRebuildReport(ctx, &out, repo, cfg, true, func() time.Time { return base }); err != nil {
		t.Fatal(err)
	}
	sessions, _ := repo.AllSessions(ctx)
	if len(sessions) != 1 {
		t.Fatalf("write must persist sessions; got %d", len(sessions))
	}
	events, _ := repo.AllEvents(ctx)
	durs, _ := repo.AllMediaDurations(ctx)
	if rebuild.Diff(sessions, rebuild.Reconstruct(events, durs, cfg)).Drift() {
		t.Fatal("drift after write")
	}
}

func TestRebuildDoesNotResurrectDeleted(t *testing.T) {
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo.FindOrCreateMediaItemWithID(ctx, "m1", "T", "vlc")
	p := 30.0
	repo.InsertEvent(ctx, store.Event{
		ID: "e1", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
		Type: "start", PositionSeconds: &p, OccurredAt: base, ReceivedAt: base, Raw: []byte("{}"),
	})
	if _, err := repo.SoftDeleteMedia(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	cfg := sessionize.Config{SessionGap: 30 * time.Minute, CompletionThreshold: 0.9, ProgressCadence: 30 * time.Second}

	var out bytes.Buffer
	if _, err := runRebuildReport(ctx, &out, repo, cfg, true, func() time.Time { return base }); err != nil {
		t.Fatal(err)
	}
	sessions, _ := repo.AllSessions(ctx)
	if len(sessions) != 0 {
		t.Fatalf("rebuild resurrected deleted media: got %d sessions", len(sessions))
	}
}
