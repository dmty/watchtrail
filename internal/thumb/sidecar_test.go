// internal/thumb/sidecar_test.go
package thumb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"watchtrail/internal/store"
)

func vlcItem(path string) store.MediaItem {
	return store.MediaItem{ID: "m1", SourceKind: "vlc", URLOrPath: "file://" + path}
}

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSidecarPriorityAndContentType(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "The Film.mkv")
	writeFile(t, video, []byte("video"))
	// Lower-priority sibling present first.
	writeFile(t, filepath.Join(dir, "The Film.png"), []byte("basename-png"))
	writeFile(t, filepath.Join(dir, "poster.jpg"), []byte("poster-bytes"))

	var s Sidecar
	item := vlcItem(video)
	if !s.Available(item) {
		t.Fatal("expected Available=true")
	}
	data, ct, ok, err := s.Resolve(context.Background(), item)
	if err != nil || !ok {
		t.Fatalf("Resolve ok=%v err=%v", ok, err)
	}
	if string(data) != "poster-bytes" {
		t.Fatalf("expected poster.jpg to win, got %q", data)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %q", ct)
	}
}

func TestSidecarNonePresent(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	writeFile(t, video, []byte("v"))
	var s Sidecar
	item := vlcItem(video)
	if s.Available(item) {
		t.Fatal("expected Available=false")
	}
	if _, _, ok, err := s.Resolve(context.Background(), item); ok || err != nil {
		t.Fatalf("expected miss, got ok=%v err=%v", ok, err)
	}
}

func TestSidecarNonLocal(t *testing.T) {
	var s Sidecar
	item := store.MediaItem{ID: "y", SourceKind: "youtube", ExternalID: "abc", URLOrPath: "https://youtu.be/abc"}
	if s.Available(item) {
		t.Fatal("non-local item should not be Available")
	}
}
