// internal/web/web.go
package web

import (
	"net/http"

	"watchtrail/internal/store"
)

// Handler builds the dashboard router over repo. Server-rendered, loopback-only,
// no auth (same posture as the read API). Returns an error if templates fail to
// parse, so the caller can fail fast at startup.
func Handler(repo store.Repository) (http.Handler, error) {
	rn, err := newRenderer()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(staticFS))
	mux.HandleFunc("GET /{$}", handleRecent(repo, rn))
	mux.HandleFunc("GET /item/{id}", handleItem(repo, rn))
	mux.HandleFunc("GET /stats", handleStats(repo, rn))
	mux.HandleFunc("GET /search", handleSearch(repo, rn))
	return mux, nil
}
