package thumb

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"watchtrail/internal/store"
)

// Chain resolves sources in order, backed by a disk cache. The cache holds
// derived, regenerable images; nothing is written to the database.
type Chain struct {
	sources  []Source
	cacheDir string
	locks    sync.Map // item id -> *sync.Mutex
}

func NewChain(cacheDir string, sources ...Source) *Chain {
	return &Chain{sources: sources, cacheDir: cacheDir}
}

// Available reports whether a thumbnail can likely be served: a cache hit, or
// any source reporting itself available. Cheap (stat-only).
func (c *Chain) Available(item store.MediaItem) bool {
	if key, ok := c.cacheKey(item); ok {
		if _, err := os.Stat(filepath.Join(c.cacheDir, key)); err == nil {
			return true
		}
	}
	for _, s := range c.sources {
		if s.Available(item) {
			return true
		}
	}
	return false
}

// Get returns a cached-or-resolved image. ok=false when no source produced one.
func (c *Chain) Get(ctx context.Context, item store.MediaItem) ([]byte, string, bool, error) {
	key, hasKey := c.cacheKey(item)
	if hasKey {
		if data, ok := c.readCache(key); ok {
			return data, http.DetectContentType(data), true, nil
		}
	}

	mu := c.lockFor(item.ID)
	mu.Lock()
	defer mu.Unlock()

	if hasKey { // re-check under lock
		if data, ok := c.readCache(key); ok {
			return data, http.DetectContentType(data), true, nil
		}
	}

	for _, s := range c.sources {
		data, ct, ok, err := s.Resolve(ctx, item)
		if err != nil {
			log.Printf("thumb: source %s: %v", s.Name(), err)
			continue
		}
		if !ok || len(data) == 0 {
			continue
		}
		if ct == "" {
			ct = http.DetectContentType(data)
		}
		if hasKey {
			c.writeCache(key, data)
		}
		return data, ct, true, nil
	}
	return nil, "", false, nil
}

func (c *Chain) lockFor(id string) *sync.Mutex {
	v, _ := c.locks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// cacheKey is id + source-file mtime + size, so replacing the file regenerates.
func (c *Chain) cacheKey(item store.MediaItem) (string, bool) {
	path, ok := LocalPath(item.URLOrPath)
	if !ok {
		return "", false
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%s.%d.%d.img", safeID(item.ID), fi.ModTime().Unix(), fi.Size()), true
}

func safeID(id string) string {
	return strings.NewReplacer("/", "_", "\\", "_", ".", "_").Replace(id)
}

func (c *Chain) readCache(key string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(c.cacheDir, key))
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func (c *Chain) writeCache(key string, data []byte) {
	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		log.Printf("thumb: cache mkdir: %v", err)
		return
	}
	dst := filepath.Join(c.cacheDir, key)
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("thumb: cache write: %v", err)
		return
	}
	if err := os.Rename(tmp, dst); err != nil {
		log.Printf("thumb: cache rename: %v", err)
	}
}
