// internal/web/item_test.go
package web

import (
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

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
