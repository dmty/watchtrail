// internal/web/events.go
package web

import (
	"fmt"
	"net/http"
	"time"

	"watchtrail/internal/events"
)

// handleEvents streams Server-Sent Events: an "update" carrying the changed media
// id on each broker ping, plus a periodic heartbeat comment. Recent re-fetches on
// any update; the Item page filters on the id. Read-only, loopback (no auth).
func handleEvents(broker *events.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, cancel := broker.Subscribe()
		defer cancel()

		fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()

		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-broker.Done():
				return
			case <-ticker.C:
				fmt.Fprint(w, ":\n\n") // heartbeat comment keeps the connection alive
				flusher.Flush()
			case id := <-ch:
				fmt.Fprintf(w, "event: update\ndata: %s\n\n", id)
				flusher.Flush()
			}
		}
	}
}
