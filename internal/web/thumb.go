// internal/web/thumb.go
package web

import (
	"log"
	"net/http"

	"watchtrail/internal/store"
	"watchtrail/internal/thumb"
)

// handleThumb serves a generated/cached thumbnail for a media item. It takes an
// item id only and reads the file path from the looked-up item, so no caller-
// supplied path ever reaches the filesystem.
func handleThumb(repo store.Repository, thumbs *thumb.Chain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if thumbs == nil {
			http.NotFound(w, r)
			return
		}
		m, ok, err := repo.MediaByID(r.Context(), r.PathValue("id"))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		data, ct, ok, err := thumbs.Get(r.Context(), m)
		if err != nil {
			log.Printf("thumb: get %s: %v", m.ID, err)
			http.NotFound(w, r)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = w.Write(data)
	}
}
