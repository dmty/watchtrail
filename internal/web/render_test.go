package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWatchedFmt(t *testing.T) {
	if got := watchedFmt(90); got != "1:30" {
		t.Fatalf("90s = %q", got)
	}
	if got := watchedFmt(3661); got != "1:01:01" {
		t.Fatalf("3661s = %q", got)
	}
}

func TestIsHTMX(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if isHTMX(r) {
		t.Fatal("plain request should not be htmx")
	}
	r.Header.Set("HX-Request", "true")
	if !isHTMX(r) {
		t.Fatal("HX-Request header should be detected")
	}
}

func TestRendererBuildsAndRendersFragment(t *testing.T) {
	rnd, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	var sb strings.Builder
	// empty rows + no cursor renders the empty-state text without error
	if err := rnd.fragment(&sb, "sessions_rows", recentFragmentData{}); err != nil {
		t.Fatalf("fragment: %v", err)
	}
	if !strings.Contains(sb.String(), "No history yet") {
		t.Fatalf("empty fragment = %q", sb.String())
	}
}
