package main

import (
	"context"
	"time"

	"watchtrail/internal/store"
)

// sessionsQuery is the CLI's filter for the sessions list (no paging in the CLI).
type sessionsQuery struct {
	From, To *time.Time
	Source   string
	Kind     string
	MediaID  string
	Limit    int
}

// itemView is a media item with its session history and rollup totals.
type itemView struct {
	ID              string
	Title           string
	Kind            string
	DurationSeconds *int
	Sessions        []store.SessionRow
	Starts          int
	Completions     int
	WatchedSeconds  int
}

// reader is the read surface the CLI renders from. Satisfied by both the HTTP
// apiClient and the direct-store storeReader so rendering has one code path.
type reader interface {
	Sessions(ctx context.Context, q sessionsQuery) ([]store.SessionRow, error)
	MediaDetail(ctx context.Context, id string) (itemView, bool, error)
	Summary(ctx context.Context, from, to *time.Time) (store.Summary, error)
}

// storeReader reads directly from the store (fallback when the API is down).
type storeReader struct {
	repo store.Repository
}

func (s *storeReader) Sessions(ctx context.Context, q sessionsQuery) ([]store.SessionRow, error) {
	rows, _, err := s.repo.Sessions(ctx, store.SessionFilter{
		From: q.From, To: q.To, Source: q.Source, Kind: q.Kind, MediaID: q.MediaID, Limit: q.Limit,
	})
	return rows, err
}

func (s *storeReader) MediaDetail(ctx context.Context, id string) (itemView, bool, error) {
	m, ok, err := s.repo.MediaByID(ctx, id)
	if err != nil || !ok {
		return itemView{}, ok, err
	}
	sessions, err := s.repo.SessionsForMedia(ctx, id)
	if err != nil {
		return itemView{}, true, err
	}
	v := itemView{
		ID: m.ID, Title: m.Title, Kind: m.Kind, DurationSeconds: m.DurationSeconds,
		Sessions: sessions, Starts: len(sessions),
	}
	if v.Title == "" {
		v.Title = m.ExternalID
	}
	for _, sess := range sessions {
		v.WatchedSeconds += sess.WatchedSeconds
		if sess.Completed {
			v.Completions++
		}
	}
	return v, true, nil
}

func (s *storeReader) Summary(ctx context.Context, from, to *time.Time) (store.Summary, error) {
	return s.repo.StatsSummary(ctx, from, to)
}
