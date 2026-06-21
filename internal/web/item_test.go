// internal/web/item_test.go
package web

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestItemYouTubeCard(t *testing.T) {
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	var mediaID string
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		id, err := r.FindOrCreateMediaItem(context.Background(), store.MediaItem{
			SourceKind:  "youtube",
			ExternalID:  "abc123_-",
			IdentityKey: "youtube:abc123_-",
			Kind:        "video",
			Title:       "Squid Game In Real Life",
			URLOrPath:   "https://www.youtube.com/watch?v=abc123_-&list=PL1&index=4",
			Language:    "es-419",
			Metadata:    json.RawMessage(`{"audio_language_label":"Spanish (Latin America)","channel":"MrBeast"}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		mediaID = id
		if err := r.UpsertSession(context.Background(), store.Session{
			ID: "s1", MediaItemID: id, SourceKind: "youtube", SourceInstance: "i1",
			StartedAt: base, EndedAt: base.Add(60 * time.Second), WatchedSeconds: 60,
			MaxPositionSeconds: 60, EventCount: 2, CreatedAt: base, UpdatedAt: base,
		}); err != nil {
			t.Fatal(err)
		}
	})

	_, body := bodyOf(t, srv.URL+"/item/"+mediaID, false)

	// Canonical watch link built from the video id — no playlist/index cruft.
	if !strings.Contains(body, `href="https://www.youtube.com/watch?v=abc123_-"`) {
		t.Fatalf("missing canonical youtube link: %q", body)
	}
	if strings.Contains(body, "list=PL1") {
		t.Fatalf("link should not carry playlist params: %q", body)
	}
	// Thumbnail keyed by video id.
	if !strings.Contains(body, "https://i.ytimg.com/vi/abc123_-/hqdefault.jpg") {
		t.Fatalf("missing thumbnail: %q", body)
	}
	// Language shown as a normalized name (es-419 -> Spanish), led on the totals line.
	if !strings.Contains(body, `class="lang-lead">Spanish<`) {
		t.Fatalf("missing language lead on totals line: %q", body)
	}
	if !strings.Contains(body, "Watch on YouTube") {
		t.Fatalf("missing watch label: %q", body)
	}
}

func TestItemDetail(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
		seedWebSession(t, r, "s2", "mX", "The Film", "vlc", base.Add(2*time.Hour), 40, false)
	})
	status, body := bodyOf(t, srv.URL+"/item/mX", false)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if !strings.Contains(body, "The Film") {
		t.Fatalf("missing title: %q", body)
	}
	if !strings.Contains(body, "started 2×, finished 1×") {
		t.Fatalf("totals line wrong: %q", body)
	}
}

func TestItemNotFound(t *testing.T) {
	srv := newWebServer(t, nil)
	status, body := bodyOf(t, srv.URL+"/item/missing", false)
	if status != 404 {
		t.Fatalf("status %d want 404", status)
	}
	if !strings.Contains(body, "Not found") {
		t.Fatalf("missing 404 page: %q", body)
	}
}

func TestItemHTMXReturnsFragmentOnly(t *testing.T) {
	base := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
	})
	status, body := bodyOf(t, srv.URL+"/item/mX", true)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if strings.Contains(body, "<html") || strings.Contains(body, "Recent") {
		t.Fatalf("htmx item request must return the fragment only: %q", body)
	}
	if !strings.Contains(body, `id="item-detail"`) || !strings.Contains(body, "watched total") {
		t.Fatalf("fragment missing the detail block: %q", body)
	}
}

func TestItemFullPageHasLiveScript(t *testing.T) {
	base := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
	})
	_, body := bodyOf(t, srv.URL+"/item/mX", false)
	if !strings.Contains(body, "new EventSource('/events')") {
		t.Fatalf("item full page should include the live script: %q", body)
	}
	if !strings.Contains(body, "mX") {
		t.Fatalf("live script should carry the media id: %q", body)
	}
}

func TestItemFragmentHasNoLiveScript(t *testing.T) {
	base := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
	})
	_, body := bodyOf(t, srv.URL+"/item/mX", true)
	if strings.Contains(body, "EventSource") {
		t.Fatalf("item fragment must not re-include the EventSource script: %q", body)
	}
}

func TestItemPageHasDeleteForm(t *testing.T) {
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
	})
	_, body := bodyOf(t, srv.URL+"/item/mX", false)
	if !strings.Contains(body, `action="/item/mX/delete"`) {
		t.Fatalf("missing delete form action: %q", body)
	}
	if !strings.Contains(body, "hx-confirm=") {
		t.Fatalf("delete form should confirm before posting: %q", body)
	}
}
