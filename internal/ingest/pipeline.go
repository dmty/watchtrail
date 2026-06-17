// Package ingest accepts events over multiple transports and converges them on a
// single pipeline that validates, normalizes, and persists each event.
package ingest

import (
	"context"
	"time"

	"watchtrail/internal/event"
	"watchtrail/internal/lang"
	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

// maxPositionSeconds clamps absurd positions (30 days). The core never trusts
// collector input but always keeps the original in raw.
const maxPositionSeconds = 30 * 24 * 3600

// Notifier receives the id of the media whose session changed, so the dashboard
// can push a targeted live update. The pipeline depends only on this interface,
// never on the concrete broker — keeping ingest decoupled from the web layer.
type Notifier interface {
	Publish(mediaID string)
}

type nopNotifier struct{}

func (nopNotifier) Publish(string) {}

// Pipeline is the single path every transport feeds. It owns no transport detail.
type Pipeline struct {
	repo   store.Repository
	sess   *sessionize.Sessionizer
	now    func() time.Time
	notify Notifier
}

// NewPipeline builds a Pipeline. now is injected for deterministic timestamps;
// cfg tunes sessionization; notify receives a ping after each sessionized event
// (pass nil to disable, e.g. for the CLI).
func NewPipeline(repo store.Repository, cfg sessionize.Config, now func() time.Time, notify Notifier) *Pipeline {
	if notify == nil {
		notify = nopNotifier{}
	}
	return &Pipeline{
		repo:   repo,
		sess:   sessionize.New(repo, cfg, now),
		now:    now,
		notify: notify,
	}
}

// Process validates one raw event, resolves its media item, persists the event
// idempotently, then assigns it to a session. raw is stored verbatim for replay.
func (p *Pipeline) Process(ctx context.Context, raw []byte) error {
	ev, err := event.Parse(raw)
	if err != nil {
		return err
	}

	mediaID, err := p.repo.FindOrCreateMediaItem(ctx, toMediaItem(ev))
	if err != nil {
		return err
	}

	stored := store.Event{
		ID:              ev.EventID,
		MediaItemID:     mediaID,
		SourceKind:      ev.SourceKind,
		SourceInstance:  ev.SourceInstance,
		Type:            ev.Type,
		PositionSeconds: clampPosition(ev.PositionSeconds),
		OccurredAt:      ev.OccurredAt,
		ReceivedAt:      p.now().UTC(),
		Raw:             raw,
	}
	if err := p.repo.InsertEvent(ctx, stored); err != nil {
		return err
	}
	// The event is already persisted (idempotent by event_id), so if sessionization
	// fails the collector can safely retry: the re-sent event dedups and re-folds.
	if _, err := p.sess.Assign(ctx, stored); err != nil {
		return err
	}
	// Session committed — ping any live dashboard with the changed media id.
	// Non-blocking; cannot fail ingest.
	p.notify.Publish(stored.MediaItemID)
	return nil
}

// toMediaItem maps a canonical event onto media-item identity. The external_id is
// taken as-is from the collector; identity_key prefixes the source kind.
func toMediaItem(ev event.Event) store.MediaItem {
	kind := ev.Media.Kind
	if kind == "" {
		kind = "unknown"
	}
	return store.MediaItem{
		SourceKind:      ev.SourceKind,
		ExternalID:      ev.Media.ExternalID,
		IdentityKey:     ev.SourceKind + ":" + ev.Media.ExternalID,
		Kind:            kind,
		Title:           ev.Media.Title,
		URLOrPath:       ev.Media.URLOrPath,
		DurationSeconds: ev.Media.DurationSeconds,
		Language:        lang.Normalize(ev.Media.Language),
		Metadata:        ev.Meta,
	}
}

func clampPosition(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	switch {
	case v < 0:
		v = 0
	case v > maxPositionSeconds:
		v = maxPositionSeconds
	}
	return &v
}
