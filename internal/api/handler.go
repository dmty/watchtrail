package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"

	"watchtrail/internal/lang"
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
	mux.HandleFunc("GET /api/v1/stats/by-language", handleStatsByLanguage(repo))
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
		from, to, err := timeRange(q)
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
			if errors.Is(err, store.ErrBadCursor) {
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

func handleStatsSummary(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		from, to, err := timeRange(q)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		s, err := repo.StatsSummary(r.Context(), from, to)
		if err != nil {
			serverErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, summaryDTO{
			From: from, To: to, WatchedSeconds: s.WatchedSeconds,
			DistinctItems: s.DistinctItems, Sessions: s.Sessions, CompletionRate: s.CompletionRate,
		})
	}
}

func handleStatsBySource(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		from, to, err := timeRange(q)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		stats, err := repo.StatsBySource(r.Context(), from, to)
		if err != nil {
			serverErr(w, err)
			return
		}
		out := make([]sourceStatDTO, 0, len(stats))
		for _, s := range stats {
			out = append(out, sourceStatDTO{
				Source: s.Source, WatchedSeconds: s.WatchedSeconds,
				Sessions: s.Sessions, CompletionRate: s.CompletionRate,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"by_source": out})
	}
}

type langStatDTO struct {
	Language       string `json:"language"` // display name, e.g. "Spanish"
	Code           string `json:"code"`     // primary subtag, e.g. "es"
	WatchedSeconds int    `json:"watched_seconds"`
}

// handleStatsByLanguage collapses the per-code watched-time rows to a primary
// language (en-US + en -> English) and returns them sorted by watched time.
func handleStatsByLanguage(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := timeRange(r.URL.Query())
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		stats, err := repo.StatsByLanguage(r.Context(), from, to)
		if err != nil {
			serverErr(w, err)
			return
		}
		byCode := make(map[string]int)
		for _, s := range stats {
			if code := lang.Primary(s.Language); code != "" {
				byCode[code] += s.WatchedSeconds
			}
		}
		out := make([]langStatDTO, 0, len(byCode))
		for code, secs := range byCode {
			out = append(out, langStatDTO{Language: lang.DisplayName(code), Code: code, WatchedSeconds: secs})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].WatchedSeconds != out[j].WatchedSeconds {
				return out[i].WatchedSeconds > out[j].WatchedSeconds
			}
			return out[i].Code < out[j].Code
		})
		writeJSON(w, http.StatusOK, map[string]any{"by_language": out})
	}
}

func handleStatsOverTime(repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		bucket := q.Get("bucket")
		if bucket == "" {
			bucket = "day"
		}
		from, to, err := timeRange(q)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		buckets, err := repo.StatsOverTime(r.Context(), bucket, from, to)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error()) // unsupported bucket = bad request
			return
		}
		out := make([]bucketDTO, 0, len(buckets))
		for _, b := range buckets {
			out = append(out, bucketDTO{Date: b.Date, WatchedSeconds: b.WatchedSeconds, Sessions: b.Sessions})
		}
		writeJSON(w, http.StatusOK, map[string]any{"buckets": out})
	}
}
