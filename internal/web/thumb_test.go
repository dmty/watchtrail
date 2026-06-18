// internal/web/thumb_test.go
package web

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"watchtrail/internal/store"
)

// seedVLCWithPoster creates a temp video + sibling poster.jpg and a VLC media
// item pointing at it; returns the item id.
func seedVLCWithPoster(t *testing.T, r *store.SQLiteRepo) string {
	t.Helper()
	dir := t.TempDir()
	video := filepath.Join(dir, "Movie.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "poster.jpg"), []byte("\xff\xd8\xff\x00POSTER"), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := r.FindOrCreateMediaItem(context.Background(), store.MediaItem{
		SourceKind:  "vlc",
		ExternalID:  "file:hash",
		IdentityKey: "vlc:file:hash",
		Kind:        "video",
		Title:       "Movie",
		URLOrPath:   "file://" + video,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestItemPageEmitsThumbForVLCWithPoster(t *testing.T) {
	var id string
	srv := newWebServer(t, func(r *store.SQLiteRepo) { id = seedVLCWithPoster(t, r) })
	_, body := bodyOf(t, srv.URL+"/item/"+id, false)
	if !strings.Contains(body, `src="/thumb/`+id+`"`) {
		t.Fatalf("item page missing /thumb img: %q", body)
	}
}

func TestThumbRouteServesPoster(t *testing.T) {
	var id string
	srv := newWebServer(t, func(r *store.SQLiteRepo) { id = seedVLCWithPoster(t, r) })
	status, body := bodyOf(t, srv.URL+"/thumb/"+id, false)
	if status != 200 || !strings.Contains(body, "POSTER") {
		t.Fatalf("thumb route status=%d body=%q", status, body)
	}
}

func TestThumbRouteMissing(t *testing.T) {
	srv := newWebServer(t, nil)
	if status, _ := bodyOf(t, srv.URL+"/thumb/nope", false); status != 404 {
		t.Fatalf("missing thumb status=%d, want 404", status)
	}
}

func TestVLCWithoutArtNoThumb(t *testing.T) {
	var id string
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		// VLC item with a non-existent path: no sidecar, file absent -> no thumb.
		var err error
		id, err = r.FindOrCreateMediaItem(context.Background(), store.MediaItem{
			SourceKind: "vlc", ExternalID: "file:x", IdentityKey: "vlc:file:x",
			Kind: "video", Title: "Gone", URLOrPath: "file:///does/not/exist.mkv",
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	_, body := bodyOf(t, srv.URL+"/item/"+id, false)
	if strings.Contains(body, `src="/thumb/`) {
		t.Fatalf("expected no thumb for art-less VLC item: %q", body)
	}
}
