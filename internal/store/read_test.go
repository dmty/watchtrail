package store

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 6, 15, 12, 30, 0, 123456789, time.UTC)
	id := "abc-123"
	cur := encodeCursor(at, id)
	if cur == "" {
		t.Fatal("empty cursor")
	}
	gotAt, gotID, err := decodeCursor(cur)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotAt.Equal(at) || gotID != id {
		t.Fatalf("got (%v,%q) want (%v,%q)", gotAt, gotID, at, id)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	if _, _, err := decodeCursor("not-base64!!"); err == nil {
		t.Fatal("expected error for malformed cursor")
	}
	if _, _, err := decodeCursor(""); err == nil {
		t.Fatal("expected error for empty cursor")
	}
}
