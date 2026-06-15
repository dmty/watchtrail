// Package rebuild reconstructs watch sessions from the raw event log, reusing the
// same gap-split rule and Fold as live sessionization so the two never diverge.
package rebuild

import (
	"sort"
	"time"

	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

// RebuiltSession is a derived session plus the ids of the events that compose it.
// Session.ID is empty until Apply mints one (live UUIDs are not reproducible).
type RebuiltSession struct {
	Session  store.Session
	EventIDs []string
}

type openGroup struct {
	events []store.Event
}

// Reconstruct groups events by (media_item_id, source_instance), splits each group
// into sessions on the gap boundary, and folds every session into its aggregate.
// Events are processed in (occurred_at, id) order. Output is sorted by
// (started_at, media_item_id, source_instance) for deterministic diffs.
func Reconstruct(events []store.Event, durations map[string]*int, cfg sessionize.Config) []RebuiltSession {
	sorted := make([]store.Event, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].OccurredAt.Equal(sorted[j].OccurredAt) {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].OccurredAt.Before(sorted[j].OccurredAt)
	})

	open := map[string]*openGroup{}
	var out []RebuiltSession
	flush := func(key string) {
		g := open[key]
		if g == nil {
			return
		}
		out = append(out, foldGroup(g.events, durations, cfg))
		delete(open, key)
	}

	for _, e := range sorted {
		key := e.MediaItemID + "\x00" + e.SourceInstance
		g := open[key]
		if g != nil {
			prevEnd := g.events[len(g.events)-1].OccurredAt
			if !sessionize.OpensNewSession(prevEnd, e.OccurredAt, cfg.SessionGap) {
				g.events = append(g.events, e)
				continue
			}
			flush(key)
		}
		open[key] = &openGroup{events: []store.Event{e}}
	}
	for key := range open {
		flush(key)
	}

	sort.SliceStable(out, func(i, j int) bool {
		si, sj := out[i].Session, out[j].Session
		if !si.StartedAt.Equal(sj.StartedAt) {
			return si.StartedAt.Before(sj.StartedAt)
		}
		if si.MediaItemID != sj.MediaItemID {
			return si.MediaItemID < sj.MediaItemID
		}
		return si.SourceInstance < sj.SourceInstance
	})
	return out
}

func foldGroup(events []store.Event, durations map[string]*int, cfg sessionize.Config) RebuiltSession {
	first := events[0]
	agg := sessionize.Fold(events, durations[first.MediaItemID], cfg)
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return RebuiltSession{
		Session: store.Session{
			MediaItemID:        first.MediaItemID,
			SourceKind:         first.SourceKind,
			SourceInstance:     first.SourceInstance,
			StartedAt:          agg.StartedAt,
			EndedAt:            agg.EndedAt,
			WatchedSeconds:     agg.WatchedSeconds,
			MaxPositionSeconds: agg.MaxPositionSeconds,
			Completed:          agg.Completed,
			EventCount:         agg.EventCount,
		},
		EventIDs: ids,
	}
}

// Change is one session whose aggregates differ between stored and rebuilt.
type Change struct {
	Key     string
	Fields  []string
	Stored  store.Session
	Rebuilt store.Session
}

// Report is the outcome of diffing stored sessions against a rebuild.
type Report struct {
	Added   []RebuiltSession // present in rebuild, absent from stored
	Removed []store.Session  // present in stored, absent from rebuild
	Changed []Change
}

// Drift reports whether the stored sessions disagree with the rebuild.
func (r Report) Drift() bool {
	return len(r.Added)+len(r.Removed)+len(r.Changed) > 0
}

func naturalKey(mediaID, instance string, started time.Time) string {
	return mediaID + "\x00" + instance + "\x00" + started.UTC().Format(time.RFC3339Nano)
}

// Diff matches stored and rebuilt sessions on the natural key
// (media_item_id, source_instance, started_at) — never on id, since live sessions
// carry UUIDs the rebuild cannot reproduce.
func Diff(stored []store.Session, rebuilt []RebuiltSession) Report {
	storedByKey := make(map[string]store.Session, len(stored))
	for _, s := range stored {
		storedByKey[naturalKey(s.MediaItemID, s.SourceInstance, s.StartedAt)] = s
	}
	var rep Report
	seen := map[string]bool{}
	for _, rb := range rebuilt {
		b := rb.Session
		key := naturalKey(b.MediaItemID, b.SourceInstance, b.StartedAt)
		seen[key] = true
		s, ok := storedByKey[key]
		if !ok {
			rep.Added = append(rep.Added, rb)
			continue
		}
		if fields := changedFields(s, b); len(fields) > 0 {
			rep.Changed = append(rep.Changed, Change{Key: key, Fields: fields, Stored: s, Rebuilt: b})
		}
	}
	for _, s := range stored {
		if !seen[naturalKey(s.MediaItemID, s.SourceInstance, s.StartedAt)] {
			rep.Removed = append(rep.Removed, s)
		}
	}
	return rep
}

func changedFields(s, b store.Session) []string {
	var f []string
	if s.WatchedSeconds != b.WatchedSeconds {
		f = append(f, "watched_seconds")
	}
	if s.MaxPositionSeconds != b.MaxPositionSeconds {
		f = append(f, "max_position_seconds")
	}
	if s.Completed != b.Completed {
		f = append(f, "completed")
	}
	if !s.EndedAt.Equal(b.EndedAt) {
		f = append(f, "ended_at")
	}
	if s.EventCount != b.EventCount {
		f = append(f, "event_count")
	}
	return f
}
