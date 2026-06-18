// internal/web/item.go
package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	Link           string // outbound link to the source (empty if none)
	LinkLabel      string // "Watch on YouTube" / "Open page"
	Thumbnail      string // poster image URL (empty if none)
	LanguageCode   string // BCP-47, upper-cased for display (empty if unknown)
	LanguageLabel  string // human label from meta, e.g. "Spanish (Latin America)"
	Starts         int
	Completions    int
	WatchedSeconds int
	Sessions       []itemSession
}

// mediaMeta is the slice of the meta blob the item card surfaces.
type mediaMeta struct {
	AudioLanguageLabel string `json:"audio_language_label"`
	AudioLanguageRaw   string `json:"audio_language_raw"`
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

// languageDisplay returns the upper-cased code and a human label for the audio
// language, drawing the label from the meta blob when present.
func languageDisplay(m store.MediaItem) (code, label string) {
	if m.Language == "" {
		return "", ""
	}
	var meta mediaMeta
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &meta)
	}
	label = meta.AudioLanguageLabel
	if label == "" {
		label = meta.AudioLanguageRaw
	}
	// Don't echo the raw code back as a "label" when it just duplicates the code.
	if strings.EqualFold(label, m.Language) {
		label = ""
	}
	return strings.ToUpper(m.Language), label
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
		code, langLabel := languageDisplay(m)
		data := itemPageData{
			ID: id, Title: m.Title, Kind: m.Kind, SourceKind: m.SourceKind,
			Link: href, LinkLabel: linkLabel, Thumbnail: thumbnail(m),
			LanguageCode: code, LanguageLabel: langLabel,
			Starts: len(sessions),
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
