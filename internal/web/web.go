// internal/web/web.go
package web

import (
	"net/http"

	"watchtrail/internal/events"
	"watchtrail/internal/store"
)

// Handler builds the dashboard router over repo. Server-rendered, loopback-only,
// no auth (same posture as the read API). broker drives the live-update SSE
// stream. Returns an error if templates fail to parse, so the caller can fail
// fast at startup.
func Handler(repo store.Repository, broker *events.Broker) (http.Handler, error) {
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
	mux.HandleFunc("GET /events", handleEvents(broker))
	return mux, nil
}
