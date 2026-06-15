package api

import (
	"time"

	"watchtrail/internal/store"
)

type sessionDTO struct {
	ID                 string    `json:"id"`
	MediaID            string    `json:"media_id"`
	Title              string    `json:"title"`
	Source             string    `json:"source"`
	StartedAt          time.Time `json:"started_at"`
	EndedAt            time.Time `json:"ended_at"`
	WatchedSeconds     int       `json:"watched_seconds"`
	MaxPositionSeconds float64   `json:"max_position_seconds"`
	Completed          bool      `json:"completed"`
	EventCount         int       `json:"event_count"`
}

type sessionsResponse struct {
	Sessions   []sessionDTO `json:"sessions"`
	NextCursor string       `json:"next_cursor"`
}

type mediaDTO struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Source          string `json:"source"`
	Kind            string `json:"kind"`
	DurationSeconds *int   `json:"duration_seconds"`
	URLOrPath       string `json:"url_or_path"`
}

type totalsDTO struct {
	Starts         int `json:"starts"`
	Completions    int `json:"completions"`
	WatchedSeconds int `json:"watched_seconds"`
}

type mediaDetailResponse struct {
	Media    mediaDTO     `json:"media"`
	Sessions []sessionDTO `json:"sessions"`
	Totals   totalsDTO    `json:"totals"`
}

type mediaSearchItemDTO struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Source          string `json:"source"`
	Kind            string `json:"kind"`
	DurationSeconds *int   `json:"duration_seconds"`
}

type summaryDTO struct {
	From           *time.Time `json:"from"`
	To             *time.Time `json:"to"`
	WatchedSeconds int        `json:"watched_seconds"`
	DistinctItems  int        `json:"distinct_items"`
	Sessions       int        `json:"sessions"`
	CompletionRate float64    `json:"completion_rate"`
}

type sourceStatDTO struct {
	Source         string  `json:"source"`
	WatchedSeconds int     `json:"watched_seconds"`
	Sessions       int     `json:"sessions"`
	CompletionRate float64 `json:"completion_rate"`
}

type bucketDTO struct {
	Date           string `json:"date"`
	WatchedSeconds int    `json:"watched_seconds"`
	Sessions       int    `json:"sessions"`
}

type healthDTO struct {
	Status   string `json:"status"`
	Events   int    `json:"events"`
	Sessions int    `json:"sessions"`
	Media    int    `json:"media"`
}

func toSessionDTO(r store.SessionRow) sessionDTO {
	return sessionDTO{
		ID: r.ID, MediaID: r.MediaItemID, Title: r.Title, Source: r.SourceKind,
		StartedAt: r.StartedAt, EndedAt: r.EndedAt, WatchedSeconds: r.WatchedSeconds,
		MaxPositionSeconds: r.MaxPositionSeconds, Completed: r.Completed, EventCount: r.EventCount,
	}
}

func toSessionDTOs(rows []store.SessionRow) []sessionDTO {
	out := make([]sessionDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toSessionDTO(r))
	}
	return out
}
