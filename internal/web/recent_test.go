// internal/web/recent_test.go
package web

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func bodyOf(t *testing.T, url string, htmx bool) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestRecentFullPage(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "m1", "Alpha Film", "vlc", base, 90, true)
	})
	status, body := bodyOf(t, srv.URL+"/", false)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if !strings.Contains(body, "<html") || !strings.Contains(body, "Alpha Film") {
		t.Fatalf("full page missing chrome/row: %q", body)
	}
	if !strings.Contains(body, `href="/item/m1"`) {
		t.Fatalf("row should link to media item: %q", body)
	}
}

func TestRecentHTMXFragmentOnly(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "m1", "Alpha Film", "vlc", base, 90, true)
	})
	status, body := bodyOf(t, srv.URL+"/", true)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if strings.Contains(body, "<html") {
		t.Fatalf("htmx fragment must not include page chrome: %q", body)
	}
	if !strings.Contains(body, "Alpha Film") || !strings.Contains(body, `id="sessions"`) {
		t.Fatalf("fragment missing rows: %q", body)
	}
}

func TestRecentSourceFilter(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "m1", "Vlc Thing", "vlc", base, 90, true)
		seedWebSession(t, r, "s2", "m2", "Tube Thing", "youtube", base.Add(time.Hour), 30, false)
	})
	_, body := bodyOf(t, srv.URL+"/?source=youtube", true)
	if strings.Contains(body, "Vlc Thing") || !strings.Contains(body, "Tube Thing") {
		t.Fatalf("source filter wrong: %q", body)
	}
}

func TestRecentMoreReturnsRowsOnlyFragment(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		for i := 0; i < recentPageSize+5; i++ {
			seedWebSession(t, r, fmt.Sprintf("s%d", i), fmt.Sprintf("m%d", i),
				fmt.Sprintf("Title %d", i), "vlc", base.Add(time.Duration(i)*time.Minute), 60, false)
		}
	})
	// initial htmx fragment has full #sessions wrapper + a More button row
	_, full := bodyOf(t, srv.URL+"/", true)
	if !strings.Contains(full, `id="sessions"`) || !strings.Contains(full, `id="more-row"`) {
		t.Fatalf("initial fragment must have #sessions wrapper and More row: %q", full)
	}
	// extract cursor and request next page — must NOT re-wrap in #sessions
	i := strings.Index(full, "cursor=")
	if i < 0 {
		t.Fatalf("missing cursor in More button: %q", full)
	}
	end := strings.IndexAny(full[i:], `&"`)
	cursor := full[i+len("cursor=") : i+end]
	_, more := bodyOf(t, srv.URL+"/?cursor="+cursor, true)
	if strings.Contains(more, `id="sessions"`) {
		t.Fatalf("paginated fragment must not include #sessions wrapper (would replace prior rows): %q", more)
	}
	if !strings.Contains(more, "<tr>") {
		t.Fatalf("paginated fragment must include row(s): %q", more)
	}
}

func TestRecentEmptyState(t *testing.T) {
	srv := newWebServer(t, nil)
	_, body := bodyOf(t, srv.URL+"/", false)
	if !strings.Contains(body, "No history yet") {
		t.Fatalf("empty state missing: %q", body)
	}
}

func TestRecentFullPageHasLiveScript(t *testing.T) {
	srv := newWebServer(t, nil)
	_, body := bodyOf(t, srv.URL+"/", false)
	if !strings.Contains(body, `EventSource("/events")`) && !strings.Contains(body, "EventSource('/events')") {
		t.Fatalf("full page should include the live-update script: %q", body)
	}
}

func TestRecentFragmentHasNoLiveScript(t *testing.T) {
	srv := newWebServer(t, nil)
	_, body := bodyOf(t, srv.URL+"/", true)
	if strings.Contains(body, "EventSource") {
		t.Fatalf("htmx fragment must not re-include the EventSource script: %q", body)
	}
}
