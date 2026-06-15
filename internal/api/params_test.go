package api

import (
	"testing"
	"time"
)

func TestParseTimeParam(t *testing.T) {
	got, err := parseTimeParam("2026-06-15")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("date parse = %v", got)
	}
	got, err = parseTimeParam("2026-06-15T12:30:00Z")
	if err != nil || got.Hour() != 12 {
		t.Fatalf("rfc parse = %v err=%v", got, err)
	}
	if _, err := parseTimeParam("nonsense"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseLimit(t *testing.T) {
	if n, err := parseLimit(""); err != nil || n != 0 {
		t.Fatalf("empty = %d err=%v", n, err)
	}
	if n, err := parseLimit("25"); err != nil || n != 25 {
		t.Fatalf("25 = %d err=%v", n, err)
	}
	if _, err := parseLimit("-1"); err == nil {
		t.Fatal("expected error for negative")
	}
	if _, err := parseLimit("abc"); err == nil {
		t.Fatal("expected error for non-numeric")
	}
}
