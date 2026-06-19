package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"watchtrail/internal/store"
)

func newAPI(t *testing.T, fn func(*store.SQLiteRepo)) *httptest.Server {
	t.Helper()
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	if fn != nil {
		fn(repo)
	}
	srv := httptest.NewServer(Handler(repo))
	t.Cleanup(func() { srv.Close(); repo.Close() })
	return srv
}

func seedAPISession(t *testing.T, r *store.SQLiteRepo, id, mediaID, title, source string, started time.Time, watched int, completed bool) {
	t.Helper()
	ctx := context.Background()
	if _, err := r.FindOrCreateMediaItemWithID(ctx, mediaID, title, source); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertSession(ctx, store.Session{
		ID: id, MediaItemID: mediaID, SourceKind: source, SourceInstance: "i1",
		StartedAt: started, EndedAt: started.Add(time.Duration(watched) * time.Second),
		WatchedSeconds: watched, MaxPositionSeconds: float64(watched), Completed: completed,
		EventCount: 2, CreatedAt: started, UpdatedAt: started,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionsEndpoint(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "m1", "Alpha", "vlc", base, 60, true)
		seedAPISession(t, r, "s2", "m2", "Beta", "vlc", base.Add(time.Hour), 30, false)
	})
	resp, err := http.Get(srv.URL + "/api/v1/sessions?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body sessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Sessions) != 2 || body.Sessions[0].ID != "s2" {
		t.Fatalf("sessions = %+v", body.Sessions)
	}
}

func TestSessionsBadCursor(t *testing.T) {
	srv := newAPI(t, nil)
	resp, err := http.Get(srv.URL + "/api/v1/sessions?cursor=!!bad")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status %d want 400", resp.StatusCode)
	}
}

func TestMediaDetailEndpoint(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "mX", "Film", "vlc", base, 100, true)
		seedAPISession(t, r, "s2", "mX", "Film", "vlc", base.Add(2*time.Hour), 40, false)
	})
	resp, err := http.Get(srv.URL + "/api/v1/media/mX")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body mediaDetailResponse
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Media.ID != "mX" || body.Totals.Starts != 2 || body.Totals.Completions != 1 || body.Totals.WatchedSeconds != 140 {
		t.Fatalf("detail = %+v", body)
	}
}

func TestMediaDetailNotFound(t *testing.T) {
	srv := newAPI(t, nil)
	resp, err := http.Get(srv.URL + "/api/v1/media/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status %d want 404", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := newAPI(t, nil)
	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var h healthDTO
	json.NewDecoder(resp.Body).Decode(&h)
	if h.Status != "ok" {
		t.Fatalf("health = %+v", h)
	}
}

func TestStatsSummaryEndpoint(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
		seedAPISession(t, r, "s2", "m2", "B", "vlc", base.Add(time.Hour), 50, false)
	})
	resp, err := http.Get(srv.URL + "/api/v1/stats/summary")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var s summaryDTO
	json.NewDecoder(resp.Body).Decode(&s)
	if s.WatchedSeconds != 150 || s.Sessions != 2 || s.DistinctItems != 2 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestStatsOverTimeBadBucket(t *testing.T) {
	srv := newAPI(t, nil)
	resp, err := http.Get(srv.URL + "/api/v1/stats/over-time?bucket=week")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status %d want 400", resp.StatusCode)
	}
}

func TestStatsOverTimeDefaultsToDay(t *testing.T) {
	orig := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = orig })
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
	})
	resp, err := http.Get(srv.URL + "/api/v1/stats/over-time")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		Buckets []bucketDTO `json:"buckets"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Buckets) != 1 || body.Buckets[0].Date != "2026-06-15" {
		t.Fatalf("buckets = %+v", body.Buckets)
	}
}

func TestSessionsBadTimeParam(t *testing.T) {
	srv := newAPI(t, nil)
	resp, err := http.Get(srv.URL + "/api/v1/sessions?from=nonsense")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("bad from status %d want 400", resp.StatusCode)
	}
	resp2, err := http.Get(srv.URL + "/api/v1/sessions?limit=-3")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 400 {
		t.Fatalf("bad limit status %d want 400", resp2.StatusCode)
	}
}

func TestStatsBySourceEndpoint(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
		seedAPISession(t, r, "s2", "m2", "B", "youtube", base.Add(time.Hour), 50, false)
	})
	resp, err := http.Get(srv.URL + "/api/v1/stats/by-source")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		BySource []sourceStatDTO `json:"by_source"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.BySource) != 2 {
		t.Fatalf("by_source = %+v", body.BySource)
	}
}

func TestStatsByLanguageEndpoint(t *testing.T) {
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		ctx := context.Background()
		seed := func(ext, langCode string, watched int) {
			id, err := r.FindOrCreateMediaItem(ctx, store.MediaItem{
				SourceKind: "youtube", ExternalID: ext, IdentityKey: "youtube:" + ext,
				Kind: "video", Title: ext, Language: langCode,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := r.UpsertSession(ctx, store.Session{
				ID: "s-" + ext, MediaItemID: id, SourceKind: "youtube", SourceInstance: "i1",
				StartedAt: base, EndedAt: base.Add(time.Duration(watched) * time.Second),
				WatchedSeconds: watched, MaxPositionSeconds: float64(watched),
				EventCount: 2, CreatedAt: base, UpdatedAt: base,
			}); err != nil {
				t.Fatal(err)
			}
		}
		seed("a", "es-419", 100)
		seed("b", "es-US", 50) // collapses with es-419 into "Spanish"
		seed("c", "en", 40)
		seed("d", "", 10) // no language -> excluded
	})
	resp, err := http.Get(srv.URL + "/api/v1/stats/by-language")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		ByLanguage []langStatDTO `json:"by_language"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.ByLanguage) != 2 {
		t.Fatalf("want 2 languages (Spanish, English), got %+v", body.ByLanguage)
	}
	top := body.ByLanguage[0]
	if top.Code != "es" || top.Language != "Spanish" || top.WatchedSeconds != 150 {
		t.Fatalf("top = %+v, want Spanish/es/150", top)
	}
	if body.ByLanguage[1].Code != "en" || body.ByLanguage[1].WatchedSeconds != 40 {
		t.Fatalf("second = %+v, want en/40", body.ByLanguage[1])
	}
}

func TestStatsSummaryBadTimeParam(t *testing.T) {
	srv := newAPI(t, nil)
	resp, err := http.Get(srv.URL + "/api/v1/stats/summary?to=garbage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status %d want 400", resp.StatusCode)
	}
}

func TestStatsOverTimeHourBucket(t *testing.T) {
	orig := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = orig })
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
	})
	resp, err := http.Get(srv.URL + "/api/v1/stats/over-time?bucket=hour")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		Buckets []bucketDTO `json:"buckets"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Buckets) != 1 || body.Buckets[0].Date != "2026-06-15T12" {
		t.Fatalf("buckets = %+v", body.Buckets)
	}
}
