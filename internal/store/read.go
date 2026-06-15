package store

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// encodeCursor packs a keyset position (started_at, id) into an opaque token.
// Loopback-only, so plain base64 of "rfc3339nano|id" is sufficient — no signing.
func encodeCursor(startedAt time.Time, id string) string {
	raw := startedAt.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cur string) (time.Time, string, error) {
	if cur == "" {
		return time.Time{}, "", errors.New("empty cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(cur)
	if err != nil {
		return time.Time{}, "", errors.New("malformed cursor")
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("malformed cursor")
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", errors.New("malformed cursor")
	}
	return at, parts[1], nil
}
