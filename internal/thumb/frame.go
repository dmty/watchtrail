// internal/thumb/frame.go
package thumb

import (
	"context"
	"net/http"
	"os"

	"watchtrail/internal/store"
)

// Extractor is the ffmpeg seam. The real implementation shells out; tests fake it.
type Extractor interface {
	Available() bool
	EmbeddedCover(ctx context.Context, path string) (data []byte, ok bool, err error)
	ExtractFrame(ctx context.Context, path string, atSeconds int) ([]byte, error)
}

// Frame produces a thumbnail from the video itself: embedded cover art if any,
// otherwise an extracted frame. Requires a working Extractor.
type Frame struct{ Ex Extractor }

func (Frame) Name() string { return "frame" }

func (f Frame) localFile(item store.MediaItem) (string, bool) {
	path, ok := LocalPath(item.URLOrPath)
	if !ok {
		return "", false
	}
	if fi, err := os.Stat(path); err != nil || fi.IsDir() {
		return "", false
	}
	return path, true
}

func (f Frame) Available(item store.MediaItem) bool {
	if f.Ex == nil || !f.Ex.Available() {
		return false
	}
	_, ok := f.localFile(item)
	return ok
}

func (f Frame) Resolve(ctx context.Context, item store.MediaItem) ([]byte, string, bool, error) {
	path, ok := f.localFile(item)
	if !ok || f.Ex == nil || !f.Ex.Available() {
		return nil, "", false, nil
	}
	if data, has, err := f.Ex.EmbeddedCover(ctx, path); err != nil {
		return nil, "", false, err
	} else if has && len(data) > 0 {
		return data, http.DetectContentType(data), true, nil
	}
	data, err := f.Ex.ExtractFrame(ctx, path, seekSeconds(item))
	if err != nil {
		return nil, "", false, err
	}
	if len(data) == 0 {
		return nil, "", false, nil
	}
	return data, "image/jpeg", true, nil
}

// seekSeconds picks min(60, 20%·duration), floored at 1; 60 when duration unknown.
func seekSeconds(item store.MediaItem) int {
	if item.DurationSeconds != nil {
		twenty := *item.DurationSeconds / 5
		if twenty < 1 {
			return 1
		}
		if twenty < 60 {
			return twenty
		}
	}
	return 60
}
