package ids

import (
	"regexp"
	"testing"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUIDFormat(t *testing.T) {
	id := NewUUID()
	if !uuidV4.MatchString(id) {
		t.Fatalf("not a v4 uuid: %q", id)
	}
}

func TestNewUUIDUnique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := NewUUID()
		if seen[id] {
			t.Fatalf("duplicate uuid: %q", id)
		}
		seen[id] = true
	}
}
