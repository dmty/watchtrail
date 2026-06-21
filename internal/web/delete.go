// internal/web/delete.go
package web

import (
	"net/http"

	"watchtrail/internal/store"
)

// handleDeleteItem soft-deletes a media item (and its sessions) then sends the
// client to Recent. htmx requests get an HX-Redirect header (200); plain form
// posts get a 303. Unknown / already-deleted ids 404.
func handleDeleteItem(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		found, err := repo.SoftDeleteMedia(r.Context(), id)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}
		if isHTMX(r) {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
