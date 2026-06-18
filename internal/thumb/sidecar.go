package thumb

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"watchtrail/internal/store"
)

var sidecarFixedNames = []string{
	"poster.jpg", "poster.png",
	"folder.jpg", "folder.png",
	"cover.jpg", "cover.png",
}

// Sidecar serves art that already sits next to the video file (Plex/Kodi/
// Jellyfin convention). No external dependency.
type Sidecar struct{}

func (Sidecar) Name() string { return "sidecar" }

func (Sidecar) candidates(videoPath string) []string {
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	out := make([]string, 0, len(sidecarFixedNames)+2)
	for _, n := range sidecarFixedNames {
		out = append(out, filepath.Join(dir, n))
	}
	return append(out, filepath.Join(dir, base+".jpg"), filepath.Join(dir, base+".png"))
}

func (s Sidecar) find(item store.MediaItem) (string, bool) {
	path, ok := LocalPath(item.URLOrPath)
	if !ok {
		return "", false
	}
	for _, c := range s.candidates(path) {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return c, true
		}
	}
	return "", false
}

func (s Sidecar) Available(item store.MediaItem) bool {
	_, ok := s.find(item)
	return ok
}

func (s Sidecar) Resolve(ctx context.Context, item store.MediaItem) ([]byte, string, bool, error) {
	c, ok := s.find(item)
	if !ok {
		return nil, "", false, nil
	}
	data, err := os.ReadFile(c)
	if err != nil {
		return nil, "", false, err
	}
	return data, contentTypeByExt(c), true, nil
}
