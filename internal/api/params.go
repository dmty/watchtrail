package api

import (
	"errors"
	"net/url"
	"strconv"
	"time"
)

// parseTimeParam accepts RFC3339 or a bare YYYY-MM-DD (interpreted as UTC midnight).
func parseTimeParam(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("invalid time: " + s)
}

// optTime parses a query param into *time.Time; absent => nil, malformed => error.
func optTime(q url.Values, key string) (*time.Time, error) {
	v := q.Get(key)
	if v == "" {
		return nil, nil
	}
	t, err := parseTimeParam(v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseLimit parses an optional positive limit; "" => 0 (handler default).
func parseLimit(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New("invalid limit")
	}
	if n < 0 {
		return 0, errors.New("limit must be >= 0")
	}
	return n, nil
}
