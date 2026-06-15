// internal/web/stats.go
package web

import (
	"net/http"

	"watchtrail/internal/store"
)

type statsPageData struct {
	WatchedSeconds int
	DistinctItems  int
	Sessions       int
	CompletionPct  float64
}

func handleStats(repo store.Repository, rn *renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := repo.StatsSummary(r.Context(), nil, nil)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = rn.page(w, "stats", statsPageData{
			WatchedSeconds: s.WatchedSeconds,
			DistinctItems:  s.DistinctItems,
			Sessions:       s.Sessions,
			CompletionPct:  s.CompletionRate * 100,
		})
	}
}
