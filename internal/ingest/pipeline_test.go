package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

func newTestPipeline(t *testing.T) (*Pipeline, store.Repository) {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	fixed := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	cfg := sessionize.Config{SessionGap: 30 * time.Minute, CompletionThreshold: 0.9, ProgressCadence: 30 * time.Second}
	return NewPipeline(repo, cfg, func() time.Time { return fixed }, nil), repo
}

type spyNotifier struct{ n int }

func (s *spyNotifier) Publish() { s.n++ }

func TestProcessPublishesAfterAssign(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	fixed := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	spy := &spyNotifier{}
	p := NewPipeline(repo, sessionize.Config{
		SessionGap: 30 * time.Minute, CompletionThreshold: 0.9, ProgressCadence: 30 * time.Second,
	}, func() time.Time { return fixed }, spy)

	raw := []byte(`{"v":1,"event_id":"e1","type":"start","occurred_at":"2026-06-16T12:00:00Z","source_kind":"vlc","media":{"external_id":"file:x","title":"X"}}`)
	if err := p.Process(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	if spy.n != 1 {
		t.Fatalf("Publish called %d times, want 1", spy.n)
	}
}

const rawEvent = `{
  "v":1,"event_id":"ev-1","type":"progress",
  "occurred_at":"2026-06-14T09:31:02Z","source_kind":"vlc",
  "source_instance":"laptop-vlc",
  "media":{"external_id":"file:abc","title":"Spirited Away","duration_seconds":7500},
  "position_seconds":1342.0,"meta":{"rate":1.0}
}`

func TestProcessPersistsEvent(t *testing.T) {
	p, repo := newTestPipeline(t)
	ctx := context.Background()
	if err := p.Process(ctx, []byte(rawEvent)); err != nil {
		t.Fatalf("Process: %v", err)
	}
	n, err := repo.CountEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("CountEvents = %d, want 1", n)
	}
}

func TestProcessIsIdempotent(t *testing.T) {
	p, repo := newTestPipeline(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := p.Process(ctx, []byte(rawEvent)); err != nil {
			t.Fatalf("Process #%d: %v", i, err)
		}
	}
	n, _ := repo.CountEvents(ctx)
	if n != 1 {
		t.Fatalf("CountEvents = %d, want 1 (same event_id)", n)
	}
}

func TestProcessRejectsInvalid(t *testing.T) {
	p, _ := newTestPipeline(t)
	if err := p.Process(context.Background(), []byte(`{"v":1,"type":"start"}`)); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestProcessClampsNegativePosition(t *testing.T) {
	p, repo := newTestPipeline(t)
	ctx := context.Background()
	body := `{"v":1,"event_id":"neg","type":"seek","occurred_at":"2026-06-14T09:31:02Z","source_kind":"vlc","media":{"external_id":"file:abc"},"position_seconds":-50.0}`
	if err := p.Process(ctx, []byte(body)); err != nil {
		t.Fatalf("Process: %v", err)
	}
	sr := repo.(*store.SQLiteRepo)
	var pos float64
	if err := sr.DB().QueryRowContext(ctx, `SELECT position_seconds FROM watch_event WHERE id='neg'`).Scan(&pos); err != nil {
		t.Fatal(err)
	}
	if pos != 0 {
		t.Fatalf("position_seconds = %v, want clamped to 0", pos)
	}
}

func TestProcessPopulatesSession(t *testing.T) {
	p, repo := newTestPipeline(t)
	ctx := context.Background()

	body := `{"v":1,"event_id":"s1","type":"start","occurred_at":"2026-06-14T09:31:02Z","source_kind":"vlc","source_instance":"laptop","media":{"external_id":"file:abc","duration_seconds":100},"position_seconds":0}`
	if err := p.Process(ctx, []byte(body)); err != nil {
		t.Fatalf("Process: %v", err)
	}

	sr := repo.(*store.SQLiteRepo)
	var sid string
	if err := sr.DB().QueryRowContext(ctx, `SELECT session_id FROM watch_event WHERE id='s1'`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if sid == "" {
		t.Fatal("event session_id is empty")
	}
	var n int
	if err := sr.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM watch_session`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("watch_session rows = %d, want 1", n)
	}
}
