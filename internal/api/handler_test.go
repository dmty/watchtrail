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
	resp, _ := http.Get(srv.URL + "/api/v1/sessions?cursor=!!bad")
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
	resp, _ := http.Get(srv.URL + "/api/v1/media/mX")
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
	resp, _ := http.Get(srv.URL + "/api/v1/media/missing")
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status %d want 404", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := newAPI(t, nil)
	resp, _ := http.Get(srv.URL + "/api/v1/health")
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
	resp, _ := http.Get(srv.URL + "/api/v1/stats/summary")
	defer resp.Body.Close()
	var s summaryDTO
	json.NewDecoder(resp.Body).Decode(&s)
	if s.WatchedSeconds != 150 || s.Sessions != 2 || s.DistinctItems != 2 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestStatsOverTimeBadBucket(t *testing.T) {
	srv := newAPI(t, nil)
	resp, _ := http.Get(srv.URL + "/api/v1/stats/over-time?bucket=week")
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status %d want 400", resp.StatusCode)
	}
}

func TestStatsOverTimeDefaultsToDay(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	srv := newAPI(t, func(r *store.SQLiteRepo) {
		seedAPISession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
	})
	resp, _ := http.Get(srv.URL + "/api/v1/stats/over-time")
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
