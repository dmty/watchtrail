// internal/thumb/build_test.go
package thumb

import "testing"

func TestBuildKnownOrder(t *testing.T) {
	c := Build(t.TempDir(), []string{"frame", "sidecar"})
	if got := sourceNames(c); !equal(got, []string{"frame", "sidecar"}) {
		t.Fatalf("sources = %v", got)
	}
}

func TestBuildDefaultsWhenEmpty(t *testing.T) {
	c := Build(t.TempDir(), nil)
	if got := sourceNames(c); !equal(got, []string{"sidecar", "frame"}) {
		t.Fatalf("default sources = %v", got)
	}
}

func TestBuildSkipsUnknown(t *testing.T) {
	c := Build(t.TempDir(), []string{"sidecar", "bogus", "frame"})
	if got := sourceNames(c); !equal(got, []string{"sidecar", "frame"}) {
		t.Fatalf("sources = %v", got)
	}
}

func sourceNames(c *Chain) []string {
	out := make([]string, 0, len(c.sources))
	for _, s := range c.sources {
		out = append(out, s.Name())
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
