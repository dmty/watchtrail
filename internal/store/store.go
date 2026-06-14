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

// Repository is the persistence boundary. Minimal surface for now.
type Repository interface {
	// FindOrCreateMediaItem returns the id of the media_item with m.IdentityKey,
	// inserting it (with a fresh id) if absent. Idempotent on identity_key.
	FindOrCreateMediaItem(ctx context.Context, m MediaItem) (string, error)
	// InsertEvent persists e, ignoring a row whose ID already exists (idempotent).
	InsertEvent(ctx context.Context, e Event) error
	// CountEvents returns the number of watch_event rows (test/health helper).
	CountEvents(ctx context.Context) (int, error)
	Close() error
}
