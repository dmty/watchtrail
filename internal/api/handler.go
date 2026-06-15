package api

import (
	"encoding/json"
	"log"
	"net/http"

	"watchtrail/internal/store"
)

// Handler builds the read-only /api/v1 router over repo. Read-only and
// loopback-only: no auth (ingestion is separately token-gated).
func Handler(repo store.Repository) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/sessions", handleSessions(repo))
	mux.HandleFunc("GET /api/v1/media/{id}", handleMediaDetail(repo))
	mux.HandleFunc("GET /api/v1/media", handleMediaSearch(repo))
	mux.HandleFunc("GET /api/v1/stats/summary", handleStatsSummary(repo))
	mux.HandleFunc("GET /api/v1/stats/by-source", handleStatsBySource(repo))
	mux.HandleFunc("GET /api/v1/stats/over-time", handleStatsOverTime(repo))
	mux.HandleFunc("GET /api/v1/health", handleHealth(repo))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// serverErr logs the real error and returns an opaque 500.
func serverErr(w http.ResponseWriter, err error) {
	log.Printf("api: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func handleSessions(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		from, err := optTime(q, "from")
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		to, err := optTime(q, "to")
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		limit, err := parseLimit(q.Get("limit"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rows, next, err := repo.Sessions(r.Context(), store.SessionFilter{
			From: from, To: to, Source: q.Get("source"), Kind: q.Get("kind"),
			MediaID: q.Get("media"), Limit: limit, Cursor: q.Get("cursor"),
		})
		if err != nil {
			if q.Get("cursor") != "" {
				writeErr(w, http.StatusBadRequest, "invalid cursor")
				return
			}
			serverErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, sessionsResponse{Sessions: toSessionDTOs(rows), NextCursor: next})
	}
}

func handleMediaDetail(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		m, ok, err := repo.MediaByID(r.Context(), id)
		if err != nil {
			serverErr(w, err)
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, "media not found")
			return
		}
		sessions, err := repo.SessionsForMedia(r.Context(), id)
		if err != nil {
			serverErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildMediaDetail(m, sessions))
	}
}

func buildMediaDetail(m store.MediaItem, sessions []store.SessionRow) mediaDetailResponse {
	totals := totalsDTO{Starts: len(sessions)}
	for _, s := range sessions {
		totals.WatchedSeconds += s.WatchedSeconds
		if s.Completed {
			totals.Completions++
		}
	}
	return mediaDetailResponse{
		Media: mediaDTO{
			ID: m.ID, Title: m.Title, Source: m.SourceKind, Kind: m.Kind,
			DurationSeconds: m.DurationSeconds, URLOrPath: m.URLOrPath,
		},
		Sessions: toSessionDTOs(sessions),
		Totals:   totals,
	}
}

func handleMediaSearch(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		items, err := repo.MediaSearch(r.Context(), q.Get("q"), q.Get("source"), q.Get("kind"))
		if err != nil {
			serverErr(w, err)
			return
		}
		out := make([]mediaSearchItemDTO, 0, len(items))
		for _, it := range items {
			out = append(out, mediaSearchItemDTO{
				ID: it.ID, Title: it.Title, Source: it.SourceKind, Kind: it.Kind,
				DurationSeconds: it.DurationSeconds,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"media": out})
	}
}

func handleHealth(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		events, err := repo.CountEvents(ctx)
		if err != nil {
			serverErr(w, err)
			return
		}
		sessions, err := repo.CountSessions(ctx)
		if err != nil {
			serverErr(w, err)
			return
		}
		media, err := repo.CountMedia(ctx)
		if err != nil {
			serverErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, healthDTO{Status: "ok", Events: events, Sessions: sessions, Media: media})
	}
}

// Temporary stub stats handlers — replaced with real implementations in the next task.
func handleStatsSummary(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { writeErr(w, http.StatusNotImplemented, "stats pending") }
}
func handleStatsBySource(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { writeErr(w, http.StatusNotImplemented, "stats pending") }
}
func handleStatsOverTime(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { writeErr(w, http.StatusNotImplemented, "stats pending") }
}
