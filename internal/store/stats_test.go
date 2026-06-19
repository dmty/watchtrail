package store

import (
	"context"
	"testing"
	"time"
)

// pinTZ fixes time.Local for the test so localtime bucketing is deterministic
// regardless of the host timezone. Restored on cleanup. Do not use with t.Parallel.
func pinTZ(t *testing.T, offsetHours int) {
	orig := time.Local
	time.Local = time.FixedZone("test", offsetHours*3600)
	t.Cleanup(func() { time.Local = orig })
}

func TestStatsSummaryAndBySource(t *testing.T) {
	r := openTemp(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	seedSession(t, r, "s1", "m1", "A", "vlc", base, 100, true)
	seedSession(t, r, "s2", "m2", "B", "vlc", base.Add(time.Hour), 50, false)
	seedSession(t, r, "s3", "m1", "A", "youtube", base.Add(2*time.Hour), 30, true)

	sum, err := r.StatsSummary(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sum.WatchedSeconds != 180 || sum.Sessions != 3 || sum.DistinctItems != 2 || sum.Completions != 2 {
		t.Fatalf("summary = %+v", sum)
	}
	if sum.CompletionRate < 0.66 || sum.CompletionRate > 0.67 {
		t.Fatalf("rate = %v", sum.CompletionRate)
	}

	bySrc, err := r.StatsBySource(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bySrc) != 2 {
		t.Fatalf("by-source len = %d", len(bySrc))
	}
}

func TestStatsOverTimeDayBuckets(t *testing.T) {
	pinTZ(t, 0) // UTC: keep the UTC-date assertions deterministic under localtime
	r := openTemp(t)
	ctx := context.Background()
	day1 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	seedSession(t, r, "s1", "m1", "A", "vlc", day1, 100, true)
	seedSession(t, r, "s2", "m2", "B", "vlc", day1.Add(time.Hour), 50, false)
	seedSession(t, r, "s3", "m1", "A", "vlc", day2, 25, false)

	buckets, err := r.StatsOverTime(ctx, "day", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 2 {
		t.Fatalf("buckets = %+v", buckets)
	}
	if buckets[0].Date != "2026-06-15" || buckets[0].WatchedSeconds != 150 || buckets[0].Sessions != 2 {
		t.Fatalf("day1 bucket = %+v", buckets[0])
	}
	if buckets[1].Date != "2026-06-16" || buckets[1].WatchedSeconds != 25 {
		t.Fatalf("day2 bucket = %+v", buckets[1])
	}
}

func TestStatsOverTimeRejectsUnknownBucket(t *testing.T) {
	r := openTemp(t)
	if _, err := r.StatsOverTime(context.Background(), "week", nil, nil); err == nil {
		t.Fatal("expected error for unsupported bucket")
	}
}

func TestStatsOverTimeHourBuckets(t *testing.T) {
	pinTZ(t, 12) // UTC+12: localtime shifts both the hour and the calendar day
	r := openTemp(t)
	ctx := context.Background()
	// 20:00Z & 20:30Z -> local 2026-06-16 08:xx ; 21:00Z -> local 09:xx
	seedSession(t, r, "s1", "m1", "A", "vlc", time.Date(2026, 6, 15, 20, 0, 0, 0, time.UTC), 100, true)
	seedSession(t, r, "s2", "m2", "B", "vlc", time.Date(2026, 6, 15, 20, 30, 0, 0, time.UTC), 50, false)
	seedSession(t, r, "s3", "m1", "A", "vlc", time.Date(2026, 6, 15, 21, 0, 0, 0, time.UTC), 25, false)

	buckets, err := r.StatsOverTime(ctx, "hour", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 2 {
		t.Fatalf("buckets = %+v", buckets)
	}
	// localtime (+12) moved 20:00Z to 08:00 the next local day — proves localization
	if buckets[0].Date != "2026-06-16T08" || buckets[0].WatchedSeconds != 150 || buckets[0].Sessions != 2 {
		t.Fatalf("hour bucket 0 = %+v", buckets[0])
	}
	if buckets[1].Date != "2026-06-16T09" || buckets[1].WatchedSeconds != 25 {
		t.Fatalf("hour bucket 1 = %+v", buckets[1])
	}
}
