package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestRenderRecent(t *testing.T) {
	rows := []store.SessionRow{{
		Title: "Demo", SourceKind: "vlc", WatchedSeconds: 90, Completed: true,
		StartedAt: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
	}}
	var buf bytes.Buffer
	if err := renderRecent(&buf, rows); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Demo") || !strings.Contains(out, "1:30") || !strings.Contains(out, "✓") {
		t.Fatalf("render = %q", out)
	}
}

func TestFormatWatched(t *testing.T) {
	if got := formatWatched(90); got != "1:30" {
		t.Fatalf("90s = %q", got)
	}
	if got := formatWatched(3661); got != "1:01:01" {
		t.Fatalf("3661s = %q", got)
	}
}
