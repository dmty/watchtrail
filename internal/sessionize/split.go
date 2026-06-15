package sessionize

import "time"

// OpensNewSession reports whether an event at evTime should start a new session
// rather than extend a prior one that ended at prevEnd. The rule is the single
// definition of the gap boundary, shared by live Assign and batch rebuild: a new
// session opens only when the event lands strictly more than gap after prevEnd.
func OpensNewSession(prevEnd, evTime time.Time, gap time.Duration) bool {
	return evTime.Sub(prevEnd) > gap
}
