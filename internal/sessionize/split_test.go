package sessionize

import (
	"testing"
	"time"
)

func TestOpensNewSession(t *testing.T) {
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	gap := 30 * time.Minute
	cases := []struct {
		name    string
		prevEnd time.Time
		evTime  time.Time
		want    bool
	}{
		{"within gap reuses", base, base.Add(10 * time.Minute), false},
		{"exactly gap reuses", base, base.Add(gap), false},
		{"beyond gap opens", base, base.Add(gap + time.Second), true},
		{"earlier event reuses", base, base.Add(-time.Minute), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := OpensNewSession(c.prevEnd, c.evTime, gap); got != c.want {
				t.Fatalf("OpensNewSession=%v want %v", got, c.want)
			}
		})
	}
}
