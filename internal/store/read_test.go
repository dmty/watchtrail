package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 6, 15, 12, 30, 0, 123456789, time.UTC)
	id := "abc-123"
	cur := encodeCursor(at, id)
	if cur == "" {
		t.Fatal("empty cursor")
	}
	gotAt, gotID, err := decodeCursor(cur)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotAt.Equal(at) || gotID != id {
		t.Fatalf("got (%v,%q) want (%v,%q)", gotAt, gotID, at, id)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	if _, _, err := decodeCursor("not-base64!!"); err == nil {
		t.Fatal("expected error for malformed cursor")
	}
	if _, _, err := decodeCursor(""); err == nil {
		t.Fatal("expected error for empty cursor")
	}
}

// seedSession inserts a media item (once per id) and a session for it.
func seedSession(t *testing.T, r *SQLiteRepo, id, mediaID, title, source string, started time.Time, watched int, completed bool) {
	t.Helper()
	ctx := context.Background()
	_, err := r.FindOrCreateMediaItemWithID(ctx, mediaID, title, source)
	if err != nil {
		t.Fatalf("seed media: %v", err)
	}
	end := started.Add(time.Duration(watched) * time.Second)
	if err := r.UpsertSession(ctx, Session{
		ID: id, MediaItemID: mediaID, SourceKind: source, SourceInstance: "i1",
		StartedAt: started, EndedAt: end, WatchedSeconds: watched,
		MaxPositionSeconds: float64(watched), Completed: completed, EventCount: 2,
		CreatedAt: started, UpdatedAt: end,
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func TestSessionsPagingAndFilters(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		seedSession(t, r, fmt.Sprintf("s%d", i), fmt.Sprintf("m%d", i),
			fmt.Sprintf("Title %d", i), "vlc", base.Add(time.Duration(i)*time.Hour), 60, i%2 == 0)
	}
	page1, next, err := r.Sessions(ctx, SessionFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || page1[0].ID != "s4" || page1[1].ID != "s3" {
		t.Fatalf("page1 ids = %v", sessionIDs(page1))
	}
	if next == "" {
		t.Fatal("expected next cursor")
	}
	page2, _, err := r.Sessions(ctx, SessionFilter{Limit: 2, Cursor: next})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || page2[0].ID != "s2" || page2[1].ID != "s1" {
		t.Fatalf("page2 ids = %v", sessionIDs(page2))
	}
	from := base.Add(3 * time.Hour)
	got, _, err := r.Sessions(ctx, SessionFilter{From: &from, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("from-filter got %d want 2", len(got))
	}
	gotM, _, err := r.Sessions(ctx, SessionFilter{MediaID: "m1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(gotM) != 1 || gotM[0].ID != "s1" {
		t.Fatalf("media-filter = %v", sessionIDs(gotM))
	}
}

func sessionIDs(rows []SessionRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}
