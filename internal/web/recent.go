// internal/web/recent.go
package web

import (
	"net/http"
	"time"

	"watchtrail/internal/store"
)

const recentPageSize = 50

type sessionRow struct {
	ID             string // media item id (link target)
	Title          string
	SourceKind     string
	StartedAt      time.Time
	WatchedSeconds int
	Completed      bool
}

type recentFilter struct {
	Source string
	From   string
	To     string
}

type recentFragmentData struct {
	Rows       []sessionRow
	NextCursor string
	Filter     recentFilter
}

type recentPageData struct {
	recentFragmentData
}

// parseDateParam parses an optional YYYY-MM-DD query param into *time.Time
// (UTC midnight). An unparseable value yields nil — bad input is ignored, not
// rejected, so the dashboard stays usable for a human.
func parseDateParam(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}

func handleRecent(repo store.Repository, rn *renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := recentFilter{Source: q.Get("source"), From: q.Get("from"), To: q.Get("to")}
		rows, next, err := repo.Sessions(r.Context(), store.SessionFilter{
			From:   parseDateParam(filter.From),
			To:     parseDateParam(filter.To),
			Source: filter.Source,
			Limit:  recentPageSize,
			Cursor: q.Get("cursor"),
		})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		data := recentFragmentData{
			Rows:       toSessionRows(rows),
			NextCursor: next,
			Filter:     filter,
		}
		if isHTMX(r) {
			_ = rn.fragment(w, "sessions_rows", data)
			return
		}
		_ = rn.page(w, "recent", recentPageData{recentFragmentData: data})
	}
}

// toSessionRows maps store rows to the view model; ID is the MEDIA id so the
// row links to /item/{mediaID}.
func toSessionRows(rows []store.SessionRow) []sessionRow {
	out := make([]sessionRow, 0, len(rows))
	for _, s := range rows {
		out = append(out, sessionRow{
			ID: s.MediaItemID, Title: s.Title, SourceKind: s.SourceKind,
			StartedAt: s.StartedAt, WatchedSeconds: s.WatchedSeconds, Completed: s.Completed,
		})
	}
	return out
}
