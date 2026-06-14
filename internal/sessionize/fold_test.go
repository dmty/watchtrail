package sessionize

import (
	"testing"
	"time"

	"watchtrail/internal/store"
)

var base = time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)

func fpos(v float64) *float64 { return &v }
func dptr(v int) *int         { return &v }

func cfg() Config {
	return Config{SessionGap: 30 * time.Minute, CompletionThreshold: 0.9, ProgressCadence: 30 * time.Second}
}

// ev builds an event at base+offsetSec with the given type and optional position.
func ev(typ string, offsetSec int, pos *float64) store.Event {
	return store.Event{Type: typ, OccurredAt: base.Add(time.Duration(offsetSec) * time.Second), PositionSeconds: pos}
}

func TestFoldEmpty(t *testing.T) {
	got := Fold(nil, nil, cfg())
	if got.EventCount != 0 || got.WatchedSeconds != 0 || got.Completed {
		t.Fatalf("empty fold = %+v", got)
	}
}

func TestFoldContinuousPlay(t *testing.T) {
	evs := []store.Event{
		ev("start", 0, fpos(0)),
		ev("progress", 30, fpos(30)),
		ev("progress", 60, fpos(60)),
		ev("progress", 90, fpos(90)),
	}
	got := Fold(evs, dptr(100), cfg())
	if got.EventCount != 4 {
		t.Errorf("EventCount = %d, want 4", got.EventCount)
	}
	if got.WatchedSeconds != 90 {
		t.Errorf("WatchedSeconds = %d, want 90", got.WatchedSeconds)
	}
	if got.MaxPositionSeconds != 90 {
		t.Errorf("MaxPositionSeconds = %v, want 90", got.MaxPositionSeconds)
	}
	if !got.StartedAt.Equal(base) || !got.EndedAt.Equal(base.Add(90*time.Second)) {
		t.Errorf("span = %v..%v", got.StartedAt, got.EndedAt)
	}
	if !got.Completed {
		t.Errorf("Completed = false, want true (90 >= 0.9*100)")
	}
}

func TestFoldPauseDoesNotCount(t *testing.T) {
	evs := []store.Event{
		ev("start", 0, fpos(0)),
		ev("progress", 30, fpos(30)),
		ev("pause", 40, fpos(40)),
		ev("resume", 340, fpos(40)), // 5 min paused
		ev("progress", 370, fpos(70)),
	}
	got := Fold(evs, dptr(1000), cfg())
	// intervals counted: 0->30 (+30), 30->40 (+10, prev=progress), 340->370 (+30, prev=resume)
	// NOT counted: 40->340 (prev=pause)
	if got.WatchedSeconds != 70 {
		t.Errorf("WatchedSeconds = %d, want 70", got.WatchedSeconds)
	}
	if got.Completed {
		t.Errorf("Completed = true, want false (70 < 900)")
	}
}

func TestFoldForwardSeekDoesNotInflate(t *testing.T) {
	evs := []store.Event{
		ev("start", 0, fpos(0)),
		ev("progress", 30, fpos(30)),
		ev("seek", 32, fpos(3000)), // jumped far ahead, only 2s of wall time
		ev("progress", 62, fpos(3030)),
	}
	got := Fold(evs, dptr(6000), cfg())
	// 0->30 (+30), 30->32 (+2, prev=progress), 32->62 (+30, prev=seek)
	if got.WatchedSeconds != 62 {
		t.Errorf("WatchedSeconds = %d, want 62", got.WatchedSeconds)
	}
	if got.MaxPositionSeconds != 3030 {
		t.Errorf("MaxPositionSeconds = %v, want 3030", got.MaxPositionSeconds)
	}
}

func TestFoldIdleGapDropped(t *testing.T) {
	evs := []store.Event{
		ev("start", 0, fpos(0)),
		ev("progress", 30, fpos(30)),
		ev("progress", 600, fpos(60)), // 570s gap > 2*cadence(60s): playback wasn't progressing
	}
	got := Fold(evs, dptr(1000), cfg())
	// 0->30 (+30), 30->600 dropped (>60s)
	if got.WatchedSeconds != 30 {
		t.Errorf("WatchedSeconds = %d, want 30", got.WatchedSeconds)
	}
}

func TestFoldUnknownDurationNeverCompleted(t *testing.T) {
	evs := []store.Event{ev("start", 0, fpos(0)), ev("progress", 30, fpos(99999))}
	got := Fold(evs, nil, cfg())
	if got.Completed {
		t.Errorf("Completed = true, want false (unknown duration)")
	}
}

func TestFoldOutOfOrderSortedFirst(t *testing.T) {
	evs := []store.Event{
		ev("progress", 60, fpos(60)),
		ev("start", 0, fpos(0)),
		ev("progress", 30, fpos(30)),
	}
	got := Fold(evs, dptr(100), cfg())
	if !got.StartedAt.Equal(base) || !got.EndedAt.Equal(base.Add(60*time.Second)) {
		t.Errorf("span = %v..%v, want sorted", got.StartedAt, got.EndedAt)
	}
	if got.WatchedSeconds != 60 {
		t.Errorf("WatchedSeconds = %d, want 60", got.WatchedSeconds)
	}
}
