// internal/web/events.go
package web

import (
	"fmt"
	"net/http"
	"time"

	"watchtrail/internal/events"
)

// handleEvents streams Server-Sent Events: a contentless "update" on each broker
// ping, plus a periodic heartbeat comment. The client (Recent page) re-fetches
// its session fragment on update. Read-only, loopback (no auth).
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
