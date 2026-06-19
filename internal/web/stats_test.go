// internal/web/stats_test.go
package web

import (
	"strings"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func TestStatsPage(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
		seedWebSession(t, r, "s2", "m2", "B", "vlc", base.Add(time.Hour), 50, false)
	})
	status, body := bodyOf(t, srv.URL+"/stats", false)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	// summary cards rendered server-side
	if !strings.Contains(body, "2:30") { // 150s watched total
		t.Fatalf("watched total card missing: %q", body)
	}
	if !strings.Contains(body, "50%") { // completion rate 1/2
		t.Fatalf("completion card missing: %q", body)
	}
	// chart canvases + Chart.js + JSON endpoints referenced
	if !strings.Contains(body, `id="overTime"`) || !strings.Contains(body, `id="bySource"`) {
		t.Fatalf("canvases missing: %q", body)
	}
	if !strings.Contains(body, "chart.js") {
		t.Fatalf("Chart.js script missing: %q", body)
	}
	if !strings.Contains(body, "/api/v1/stats/over-time") || !strings.Contains(body, "/api/v1/stats/by-source") {
		t.Fatalf("chart data endpoints missing: %q", body)
	}
}

func TestStatsPageRangeWidget(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
	})
	status, body := bodyOf(t, srv.URL+"/stats", false)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	for _, want := range []string{
		`class="range"`, `id="rangeLabel"`,
		`data-step="-1"`, `data-step="1"`,
		`id="cWatched"`, `id="cItems"`, `id="cSessions"`, `id="cDone"`,
		`id="bySourceEmpty"`, `id="byLangEmpty"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in stats page: %q", want, body)
		}
	}
}
