// internal/thumb/frame_test.go
package thumb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"watchtrail/internal/store"
)

type fakeExtractor struct {
	available bool
	cover     []byte
	hasCover  bool
	frame     []byte
	gotSeek   int
}

func (f *fakeExtractor) Available() bool { return f.available }
func (f *fakeExtractor) EmbeddedCover(ctx context.Context, path string) ([]byte, bool, error) {
	return f.cover, f.hasCover, nil
}
func (f *fakeExtractor) ExtractFrame(ctx context.Context, path string, at int) ([]byte, error) {
	f.gotSeek = at
	return f.frame, nil
}

func tempVideo(t *testing.T) store.MediaItem {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "v.mkv")
	if err := os.WriteFile(p, []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}
	return store.MediaItem{ID: "m1", SourceKind: "vlc", URLOrPath: "file://" + p}
}

func TestFrameEmbeddedCoverWins(t *testing.T) {
	ex := &fakeExtractor{available: true, hasCover: true, cover: []byte("\xff\xd8\xff\x00cover")}
	f := Frame{Ex: ex}
	item := tempVideo(t)
	data, ct, ok, err := f.Resolve(context.Background(), item)
	if err != nil || !ok || string(data) != "\xff\xd8\xff\x00cover" {
		t.Fatalf("cover not returned: ok=%v err=%v data=%q", ok, err, data)
	}
	if ct != "image/jpeg" { // sniffed from JPEG magic bytes
		t.Fatalf("content type = %q", ct)
	}
}

func TestFrameExtractFallback(t *testing.T) {
	ex := &fakeExtractor{available: true, hasCover: false, frame: []byte("frame")}
	f := Frame{Ex: ex}
	dur := 1000
	item := tempVideo(t)
	item.DurationSeconds = &dur
	data, ct, ok, err := f.Resolve(context.Background(), item)
	if err != nil || !ok || string(data) != "frame" || ct != "image/jpeg" {
		t.Fatalf("frame not returned: ok=%v err=%v data=%q ct=%q", ok, err, data, ct)
	}
	if ex.gotSeek != 60 { // 20% of 1000 = 200, capped at 60
		t.Fatalf("seek = %d, want 60", ex.gotSeek)
	}
}

func TestFrameUnavailable(t *testing.T) {
	f := Frame{Ex: &fakeExtractor{available: false}}
	item := tempVideo(t)
	if f.Available(item) {
		t.Fatal("expected Available=false when extractor unavailable")
	}
	if _, _, ok, _ := f.Resolve(context.Background(), item); ok {
		t.Fatal("expected ok=false when extractor unavailable")
	}
}

func TestSeekSeconds(t *testing.T) {
	mk := func(d *int) store.MediaItem { return store.MediaItem{DurationSeconds: d} }
	n := func(v int) *int { return &v }
	for _, c := range []struct {
		dur  *int
		want int
	}{
		{nil, 60}, {n(7200), 60}, {n(100), 20}, {n(3), 1},
	} {
		if got := seekSeconds(mk(c.dur)); got != c.want {
			t.Errorf("seekSeconds(dur=%v) = %d, want %d", c.dur, got, c.want)
		}
	}
}
