package sessionize

import (
	"context"
	"time"

	"watchtrail/internal/ids"
	"watchtrail/internal/store"
)

// Store is the persistence surface the Sessionizer needs.
type Store interface {
	LatestSessionFor(ctx context.Context, mediaItemID, sourceInstance string) (store.Session, bool, error)
	SetEventSession(ctx context.Context, eventID, sessionID string) error
	EventsForSession(ctx context.Context, sessionID string) ([]store.Event, error)
	UpsertSession(ctx context.Context, s store.Session) error
	MediaDuration(ctx context.Context, mediaItemID string) (*int, error)
}

// Sessionizer assigns events to sessions lazily, as they arrive.
type Sessionizer struct {
	store Store
	cfg   Config
	now   func() time.Time
}

// New builds a Sessionizer. now is injected for deterministic timestamps.
func New(s Store, cfg Config, now func() time.Time) *Sessionizer {
	return &Sessionizer{store: s, cfg: cfg, now: now}
}

// Assign places an already-persisted event into a session: it reuses the latest
// session for the (media, instance) key when the event falls within the gap,
// otherwise opens a new one. It backfills the event's session_id, then re-folds
// the session's events and upserts the row. Returns the session id.
func (s *Sessionizer) Assign(ctx context.Context, ev store.Event) (string, error) {
	latest, found, err := s.store.LatestSessionFor(ctx, ev.MediaItemID, ev.SourceInstance)
	if err != nil {
		return "", err
	}

	var sessionID string
	var createdAt time.Time
	if found && ev.OccurredAt.Sub(latest.EndedAt) <= s.cfg.SessionGap {
		sessionID = latest.ID
		createdAt = latest.CreatedAt
	} else {
		sessionID = ids.NewUUID()
		createdAt = s.now().UTC()
	}

	if err := s.store.SetEventSession(ctx, ev.ID, sessionID); err != nil {
		return "", err
	}

	events, err := s.store.EventsForSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	duration, err := s.store.MediaDuration(ctx, ev.MediaItemID)
	if err != nil {
		return "", err
	}
	agg := Fold(events, duration, s.cfg)

	return sessionID, s.store.UpsertSession(ctx, store.Session{
		ID:                 sessionID,
		MediaItemID:        ev.MediaItemID,
		SourceKind:         ev.SourceKind,
		SourceInstance:     ev.SourceInstance,
		StartedAt:          agg.StartedAt,
		EndedAt:            agg.EndedAt,
		WatchedSeconds:     agg.WatchedSeconds,
		MaxPositionSeconds: agg.MaxPositionSeconds,
		Completed:          agg.Completed,
		EventCount:         agg.EventCount,
		CreatedAt:          createdAt,
		UpdatedAt:          s.now().UTC(),
	})
}
