// Package ingest accepts events over multiple transports and converges them on a
// single pipeline that validates, normalizes, and persists each event.
package ingest

import (
	"context"
	"time"

	"watchtrail/internal/event"
	"watchtrail/internal/store"
)

// maxPositionSeconds clamps absurd positions (30 days). The core never trusts
// collector input but always keeps the original in raw.
const maxPositionSeconds = 30 * 24 * 3600

// Pipeline is the single path every transport feeds. It owns no transport detail.
type Pipeline struct {
	repo store.Repository
	now  func() time.Time
}

// NewPipeline builds a Pipeline. now is injected for deterministic received_at.
func NewPipeline(repo store.Repository, now func() time.Time) *Pipeline {
	return &Pipeline{repo: repo, now: now}
}

// Process validates one raw event, resolves its media item, and persists the
// event idempotently. raw is stored verbatim for future replay.
func (p *Pipeline) Process(ctx context.Context, raw []byte) error {
	ev, err := event.Parse(raw)
	if err != nil {
		return err
	}

	mediaID, err := p.repo.FindOrCreateMediaItem(ctx, toMediaItem(ev))
	if err != nil {
		return err
	}

	return p.repo.InsertEvent(ctx, store.Event{
		ID:              ev.EventID,
		MediaItemID:     mediaID,
		SourceKind:      ev.SourceKind,
		SourceInstance:  ev.SourceInstance,
		Type:            ev.Type,
		PositionSeconds: clampPosition(ev.PositionSeconds),
		OccurredAt:      ev.OccurredAt,
		ReceivedAt:      p.now().UTC(),
		Raw:             raw,
	})
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
