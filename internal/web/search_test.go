// internal/web/search_test.go
package web

import (
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestSearchFullPageAndFragment(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "m1", "Cosmos Episode 1", "vlc", base, 60, false)
		seedWebSession(t, r, "s2", "m2", "Breaking News", "youtube", base, 60, false)
	})
	// full page with a query returns chrome + the matching item
	status, body := bodyOf(t, srv.URL+"/search?q=cosmos", false)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if !strings.Contains(body, "<html") || !strings.Contains(body, "Cosmos Episode 1") {
		t.Fatalf("full search page wrong: %q", body)
	}
	if !strings.Contains(body, `href="/item/m1"`) {
		t.Fatalf("result should link to item: %q", body)
	}
	// htmx request returns only the results fragment
	_, frag := bodyOf(t, srv.URL+"/search?q=cosmos", true)
	if strings.Contains(frag, "<html") {
		t.Fatalf("htmx search must be fragment-only: %q", frag)
	}
	if !strings.Contains(frag, `id="results"`) || !strings.Contains(frag, "Cosmos") {
		t.Fatalf("fragment missing results: %q", frag)
	}
}

func TestSearchEmptyQueryPrompts(t *testing.T) {
	srv := newWebServer(t, func(r *store.SQLiteRepo) {})
	_, body := bodyOf(t, srv.URL+"/search", false)
	if !strings.Contains(body, "Type to search") {
		t.Fatalf("empty search should prompt: %q", body)
	}
}
