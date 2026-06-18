// internal/web/web_test.go
package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"watchtrail/internal/events"
	"watchtrail/internal/store"
	"watchtrail/internal/thumb"
)

// newWebServer spins an httptest server over the dashboard, seeded by fn.
func newWebServer(t *testing.T, fn func(*store.SQLiteRepo)) *httptest.Server {
	t.Helper()
	repo, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	if fn != nil {
		fn(repo)
	}
	h, err := Handler(repo, events.New(), thumb.Build(t.TempDir(), nil))
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(func() { srv.Close(); repo.Close() })
	return srv
}

func seedWebSession(t *testing.T, r *store.SQLiteRepo, id, mediaID, title, source string, started time.Time, watched int, completed bool) {
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

func TestStaticAssetServed(t *testing.T) {
	srv := newWebServer(t, nil)
	resp, err := http.Get(srv.URL + "/static/htmx.min.js")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("htmx asset status %d", resp.StatusCode)
	}
}
