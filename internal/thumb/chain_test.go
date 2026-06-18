// internal/thumb/chain_test.go
package thumb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"watchtrail/internal/store"
)

type stubSource struct {
	name  string
	avail bool
	data  []byte
	ct    string
	ok    bool
	err   error
	calls int
}

func (s *stubSource) Name() string                       { return s.name }
func (s *stubSource) Available(store.MediaItem) bool      { return s.avail }
func (s *stubSource) Resolve(context.Context, store.MediaItem) ([]byte, string, bool, error) {
	s.calls++
	return s.data, s.ct, s.ok, s.err
}

func chainItem(t *testing.T) store.MediaItem {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "v.mkv")
	if err := os.WriteFile(p, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	return store.MediaItem{ID: "m1", SourceKind: "vlc", URLOrPath: "file://" + p}
}

func TestChainOrderAndErrorSkip(t *testing.T) {
	bad := &stubSource{name: "bad", err: errors.New("boom")}
	good := &stubSource{name: "good", data: []byte("img"), ct: "image/png", ok: true}
	c := NewChain(t.TempDir(), bad, good)
	data, ct, ok, err := c.Get(context.Background(), chainItem(t))
	if err != nil || !ok || string(data) != "img" || ct != "image/png" {
		t.Fatalf("Get = (%q,%q,%v,%v)", data, ct, ok, err)
	}
	if bad.calls != 1 || good.calls != 1 {
		t.Fatalf("calls bad=%d good=%d", bad.calls, good.calls)
	}
}

func TestChainCachesAfterFirstResolve(t *testing.T) {
	src := &stubSource{name: "s", data: []byte("\x89PNG\r\n\x1a\ndata"), ct: "image/png", ok: true}
	c := NewChain(t.TempDir(), src)
	item := chainItem(t)
	if _, _, ok, _ := c.Get(context.Background(), item); !ok {
		t.Fatal("first Get miss")
	}
	if _, ct, ok, _ := c.Get(context.Background(), item); !ok || ct != "image/png" {
		t.Fatalf("second Get ok=%v ct=%q", ok, ct)
	}
	if src.calls != 1 {
		t.Fatalf("source called %d times, want 1 (cache hit on second)", src.calls)
	}
}

func TestChainTotalMiss(t *testing.T) {
	c := NewChain(t.TempDir(), &stubSource{name: "s", ok: false})
	if _, _, ok, err := c.Get(context.Background(), chainItem(t)); ok || err != nil {
		t.Fatalf("expected miss, got ok=%v err=%v", ok, err)
	}
}

func TestChainAvailable(t *testing.T) {
	c := NewChain(t.TempDir(), &stubSource{name: "s", avail: true})
	if !c.Available(chainItem(t)) {
		t.Fatal("expected Available=true via source")
	}
	c2 := NewChain(t.TempDir(), &stubSource{name: "s", avail: false})
	if c2.Available(chainItem(t)) {
		t.Fatal("expected Available=false")
	}
}
