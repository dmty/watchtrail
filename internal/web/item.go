// internal/web/item.go
package web

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"watchtrail/internal/lang"
	"watchtrail/internal/store"
)

type itemSession struct {
	StartedAt      time.Time
	WatchedSeconds int
	Completed      bool
}

type itemPageData struct {
	ID             string
	Title          string
	Kind           string
	SourceKind     string
	Link            string // outbound link to the source (empty if none)
	LinkLabel       string // "Watch on YouTube" / "Open page"
	Thumbnail       string // poster image URL (empty if none)
	LanguageDisplay string // normalized language name, e.g. "Spanish" (empty if unknown)
	Starts          int
	Completions     int
	WatchedSeconds  int
	Sessions        []itemSession
}

// link returns the outbound URL and its label for a media item, by source.
// YouTube gets a canonical watch URL built from the video id (no playlist/index
// cruft); web reuses the captured page URL; other sources (e.g. local VLC files)
// get no web link.
func link(m store.MediaItem) (href, label string) {
	switch m.SourceKind {
	case "youtube":
		if m.ExternalID != "" {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(m.ExternalID), "Watch on YouTube"
		}
	case "web":
		if strings.HasPrefix(m.URLOrPath, "http://") || strings.HasPrefix(m.URLOrPath, "https://") {
			return m.URLOrPath, "Open page"
		}
	}
	return "", ""
}

// thumbnail returns a poster image URL for the media, or "" when none exists.
// YouTube exposes a stable thumbnail endpoint keyed by video id.
func thumbnail(m store.MediaItem) string {
	if m.SourceKind == "youtube" && m.ExternalID != "" {
		return "https://i.ytimg.com/vi/" + url.PathEscape(m.ExternalID) + "/hqdefault.jpg"
	}
	return ""
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
		href, linkLabel := link(m)
		data := itemPageData{
			ID: id, Title: m.Title, Kind: m.Kind, SourceKind: m.SourceKind,
			Link: href, LinkLabel: linkLabel, Thumbnail: thumbnail(m),
			LanguageDisplay: lang.DisplayName(m.Language),
			Starts:          len(sessions),
		}
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
		if isHTMX(r) {
			_ = rn.fragment(w, "item_detail", data)
			return
		}
		_ = rn.page(w, "item", data)
	}
}
