// internal/thumb/thumb.go
package thumb

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"

	"watchtrail/internal/store"
)

// Source resolves a thumbnail image for a media item. Implementations are
// composed into a Chain and tried in order.
type Source interface {
	Name() string
	// Available is a cheap (stat-only) check of whether this source can likely
	// produce a thumbnail. It must not run ffmpeg or any heavy work.
	Available(item store.MediaItem) bool
	// Resolve returns image bytes and a content type. ok=false defers to the
	// next source; a non-nil err is logged and also defers.
	Resolve(ctx context.Context, item store.MediaItem) (data []byte, contentType string, ok bool, err error)
}

// LocalPath converts a stored url_or_path into an on-disk path. VLC captures
// file:// uris; a bare absolute path is also accepted defensively. Non-file
// values (url:, http(s)://, empty) return ok=false.
func LocalPath(urlOrPath string) (string, bool) {
	switch {
	case urlOrPath == "":
		return "", false
	case strings.HasPrefix(urlOrPath, "file://"):
		u, err := url.Parse(urlOrPath)
		if err != nil || u.Path == "" {
			return "", false
		}
		return u.Path, true // url.Parse percent-decodes into u.Path
	case strings.HasPrefix(urlOrPath, "/"):
		return urlOrPath, true
	default:
		return "", false
	}
}

func contentTypeByExt(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}
