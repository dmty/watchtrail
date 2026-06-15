// Package store hides persistence behind the Repository interface so storage can
// evolve (Postgres, sync-replicated) without touching business logic.
package store

import (
	"context"
	"encoding/json"
	"time"
)

// MediaItem is the deduplicated identity of a watched thing.
type MediaItem struct {
	ID              string
	SourceKind      string
	ExternalID      string
	IdentityKey     string // source_kind + ":" + external_id, unique
	Kind            string
	Title           string
	URLOrPath       string
	DurationSeconds *int
	Metadata        json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Event is a persisted watch_event row.
type Event struct {
	ID              string // == collector event_id; PK; idempotency key
	MediaItemID     string
	SourceKind      string
	SourceInstance  string
	Type            string
	PositionSeconds *float64
	OccurredAt      time.Time
	ReceivedAt      time.Time
	Raw             json.RawMessage
}

// Session mirrors a watch_session row (a derived viewing).
type Session struct {
	ID                 string
	MediaItemID        string
	SourceKind         string
	SourceInstance     string
	StartedAt          time.Time
	EndedAt            time.Time
	WatchedSeconds     int
	MaxPositionSeconds float64
	Completed          bool
	EventCount         int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Repository is the persistence boundary. Minimal surface for now.
type Repository interface {
	// FindOrCreateMediaItem returns the id of the media_item with m.IdentityKey,
	// inserting it (with a fresh id) if absent. Idempotent on identity_key.
	FindOrCreateMediaItem(ctx context.Context, m MediaItem) (string, error)
	// InsertEvent persists e, ignoring a row whose ID already exists (idempotent).
	InsertEvent(ctx context.Context, e Event) error
	// CountEvents returns the number of watch_event rows (test/health helper).
	CountEvents(ctx context.Context) (int, error)
	// MediaDuration returns the media item's duration in seconds, or nil if unknown.
	MediaDuration(ctx context.Context, mediaItemID string) (*int, error)
	// LatestSessionFor returns the most recent session for a media/instance key.
	LatestSessionFor(ctx context.Context, mediaItemID, sourceInstance string) (Session, bool, error)
	// SetEventSession backfills a stored event's session_id.
	SetEventSession(ctx context.Context, eventID, sessionID string) error
	// EventsForSession returns a session's events ordered by occurred_at.
	EventsForSession(ctx context.Context, sessionID string) ([]Event, error)
	// UpsertSession inserts or updates a session row (created_at preserved on update).
	UpsertSession(ctx context.Context, s Session) error

	// Read API surface (M2).
	Sessions(ctx context.Context, f SessionFilter) ([]SessionRow, string, error)
	MediaByID(ctx context.Context, id string) (MediaItem, bool, error)
	SessionsForMedia(ctx context.Context, mediaID string) ([]SessionRow, error)
	MediaSearch(ctx context.Context, q, source, kind string) ([]MediaItemSummary, error)
	StatsSummary(ctx context.Context, from, to *time.Time) (Summary, error)
	StatsBySource(ctx context.Context, from, to *time.Time) ([]SourceStat, error)
	StatsOverTime(ctx context.Context, bucket string, from, to *time.Time) ([]Bucket, error)
	CountSessions(ctx context.Context) (int, error)
	CountMedia(ctx context.Context) (int, error)

	// Replay surface (M2 rebuild-sessions).
	AllEvents(ctx context.Context) ([]Event, error)
	AllSessions(ctx context.Context) ([]Session, error)
	AllMediaDurations(ctx context.Context) (map[string]*int, error)
	ReplaceAllSessions(ctx context.Context, writes []SessionWrite) error

	Close() error
}
