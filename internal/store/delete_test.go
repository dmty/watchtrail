package store

import (
	"context"
	"testing"
	"time"
)

func TestSoftDeleteMediaCascadesAndHides(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	if _, err := r.FindOrCreateMediaItemWithID(ctx, "m1", "The Film", "vlc"); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertSession(ctx, Session{
		ID: "s1", MediaItemID: "m1", SourceKind: "vlc", SourceInstance: "i1",
		StartedAt: base, EndedAt: base.Add(60 * time.Second), WatchedSeconds: 60,
		MaxPositionSeconds: 60, EventCount: 2, CreatedAt: base, UpdatedAt: base,
	}); err != nil {
		t.Fatal(err)
	}

	found, err := r.SoftDeleteMedia(ctx, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true deleting a live item")
	}

	if _, ok, err := r.MediaByID(ctx, "m1"); err != nil || ok {
		t.Fatalf("MediaByID after delete: ok=%v err=%v want ok=false", ok, err)
	}
	rows, err := r.SessionsForMedia(ctx, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("sessions after delete = %d want 0 (cascade)", len(rows))
	}
}

func TestSoftDeleteMediaIdempotentAndUnknown(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	if _, err := r.FindOrCreateMediaItemWithID(ctx, "m1", "T", "vlc"); err != nil {
		t.Fatal(err)
	}
	if found, err := r.SoftDeleteMedia(ctx, "m1"); err != nil || !found {
		t.Fatalf("first delete found=%v err=%v", found, err)
	}
	if found, err := r.SoftDeleteMedia(ctx, "m1"); err != nil || found {
		t.Fatalf("second delete found=%v err=%v want found=false", found, err)
	}
	if found, err := r.SoftDeleteMedia(ctx, "nope"); err != nil || found {
		t.Fatalf("unknown delete found=%v err=%v want found=false", found, err)
	}
}
