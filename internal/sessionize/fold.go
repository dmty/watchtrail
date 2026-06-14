// Package sessionize derives watch sessions from the append-only event stream.
// Fold is the single definition of session semantics; it powers both live
// incremental sessionization and full event-log replay.
package sessionize

import (
	"math"
	"sort"
	"time"

	"watchtrail/internal/store"
)

// Config carries the session tunables, sourced from service config.
type Config struct {
	SessionGap          time.Duration // max gap between events in one session
	CompletionThreshold float64       // fraction of duration that counts as completed
	ProgressCadence     time.Duration // expected gap between progress events
}

// SessionAgg is the derived summary of one session's events.
type SessionAgg struct {
	StartedAt          time.Time
	EndedAt            time.Time
	WatchedSeconds     int
	MaxPositionSeconds float64
	Completed          bool
	EventCount         int
}

// Fold reduces one session's events into its aggregate. Events need not be
// pre-sorted; a copy is sorted by occurred_at. durationSeconds is the media's
// known length (nil/unknown => never completed). watched_seconds is "time
// present": wall-clock between consecutive playback events, excluding paused
// intervals and idle gaps longer than twice the progress cadence.
func Fold(events []store.Event, durationSeconds *int, cfg Config) SessionAgg {
	if len(events) == 0 {
		return SessionAgg{}
	}
	evs := make([]store.Event, len(events))
	copy(evs, events)
	sort.SliceStable(evs, func(i, j int) bool {
		return evs[i].OccurredAt.Before(evs[j].OccurredAt)
	})

	agg := SessionAgg{
		StartedAt:  evs[0].OccurredAt,
		EndedAt:    evs[len(evs)-1].OccurredAt,
		EventCount: len(evs),
	}

	var watched time.Duration
	maxGap := 2 * cfg.ProgressCadence
	for i, e := range evs {
		if e.PositionSeconds != nil {
			if p := max(*e.PositionSeconds, 0); p > agg.MaxPositionSeconds {
				agg.MaxPositionSeconds = p
			}
		}
		if i == 0 {
			continue
		}
		prev := evs[i-1]
		if prev.Type == "pause" || prev.Type == "stop" {
			continue // not playing across this interval
		}
		if d := e.OccurredAt.Sub(prev.OccurredAt); d > 0 && d <= maxGap {
			watched += d
		}
	}
	agg.WatchedSeconds = int(math.Round(watched.Seconds()))

	if durationSeconds != nil && *durationSeconds > 0 &&
		agg.MaxPositionSeconds >= cfg.CompletionThreshold*float64(*durationSeconds) {
		agg.Completed = true
	}
	return agg
}
