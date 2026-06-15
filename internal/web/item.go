// internal/web/item.go
package web

import (
	"net/http"
	"time"

	"watchtrail/internal/store"
)

type itemSession struct {
	StartedAt      time.Time
	WatchedSeconds int
	Completed      bool
}

type itemPageData struct {
	Title          string
	Kind           string
	Starts         int
	Completions    int
	WatchedSeconds int
	Sessions       []itemSession
}

func handleItem(repo store.Repository, rn *renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		m, ok, err := repo.MediaByID(r.Context(), id)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = rn.page(w, "not_found", "no media item "+id)
			return
		}
		sessions, err := repo.SessionsForMedia(r.Context(), id)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		data := itemPageData{Title: m.Title, Kind: m.Kind, Starts: len(sessions)}
		if data.Title == "" {
			data.Title = m.ExternalID
		}
		for _, s := range sessions {
			data.WatchedSeconds += s.WatchedSeconds
			if s.Completed {
				data.Completions++
			}
			data.Sessions = append(data.Sessions, itemSession{
				StartedAt: s.StartedAt, WatchedSeconds: s.WatchedSeconds, Completed: s.Completed,
			})
		}
		_ = rn.page(w, "item", data)
	}
}
