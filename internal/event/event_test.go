package event

import (
	"errors"
	"testing"
	"time"
)

const valid = `{
  "v": 1,
  "event_id": "11111111-1111-4111-8111-111111111111",
  "type": "progress",
  "occurred_at": "2026-06-14T09:31:02Z",
  "source_kind": "vlc",
  "source_instance": "laptop-vlc",
  "media": { "external_id": "file:abc", "title": "Spirited Away", "duration_seconds": 7500 },
  "position_seconds": 1342.0,
  "meta": { "rate": 1.0 }
}`

func TestParseValid(t *testing.T) {
	ev, err := Parse([]byte(valid))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ev.EventID != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("EventID = %q", ev.EventID)
	}
	if ev.Type != "progress" || ev.SourceKind != "vlc" {
		t.Errorf("unexpected fields: %+v", ev)
	}
	if ev.Media.ExternalID != "file:abc" {
		t.Errorf("ExternalID = %q", ev.Media.ExternalID)
	}
	if ev.PositionSeconds == nil || *ev.PositionSeconds != 1342.0 {
		t.Errorf("PositionSeconds = %v", ev.PositionSeconds)
	}
}

func TestParseRejectsUnknownVersion(t *testing.T) {
	body := `{"v":2,"event_id":"x","type":"start","occurred_at":"2026-06-14T09:31:02Z","source_kind":"vlc","media":{"external_id":"file:abc"}}`
	_, err := Parse([]byte(body))
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("err = %v, want ErrUnsupportedVersion", err)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	cases := map[string]Event{
		"no event_id":    {V: 1, Type: "start", SourceKind: "vlc", OccurredAt: mustTime(), Media: Media{ExternalID: "file:abc"}},
		"no type":        {V: 1, EventID: "x", SourceKind: "vlc", OccurredAt: mustTime(), Media: Media{ExternalID: "file:abc"}},
		"bad type":       {V: 1, EventID: "x", Type: "explode", SourceKind: "vlc", OccurredAt: mustTime(), Media: Media{ExternalID: "file:abc"}},
		"no source_kind": {V: 1, EventID: "x", Type: "start", OccurredAt: mustTime(), Media: Media{ExternalID: "file:abc"}},
		"no external_id": {V: 1, EventID: "x", Type: "start", SourceKind: "vlc", OccurredAt: mustTime()},
		"zero occurred":  {V: 1, EventID: "x", Type: "start", SourceKind: "vlc", Media: Media{ExternalID: "file:abc"}},
	}
	for name, ev := range cases {
		if err := ev.Validate(); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: err = %v, want ErrValidation", name, err)
		}
	}
}

func TestValidateAcceptsMinimal(t *testing.T) {
	ev := Event{V: 1, EventID: "x", Type: "start", SourceKind: "vlc", OccurredAt: mustTime(), Media: Media{ExternalID: "file:abc"}}
	if err := ev.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func mustTime() time.Time {
	tm, err := time.Parse(time.RFC3339, "2026-06-14T09:31:02Z")
	if err != nil {
		panic(err)
	}
	return tm
}
