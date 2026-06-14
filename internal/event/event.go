// Package event defines the canonical watch-event wire contract shared by every
// collector, plus parsing and validation.
package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Version is the protocol major version this build accepts.
const Version = 1

// Sentinel errors so transports can map to HTTP status codes.
var (
	ErrUnsupportedVersion = errors.New("unsupported protocol version")
	ErrValidation         = errors.New("event validation failed")
)

// Media is the source-supplied identity/metadata for the thing watched.
type Media struct {
	ExternalID      string `json:"external_id"`
	Kind            string `json:"kind,omitempty"`
	Title           string `json:"title,omitempty"`
	URLOrPath       string `json:"url_or_path,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
}

// Event is one canonical watch event.
type Event struct {
	V               int             `json:"v"`
	EventID         string          `json:"event_id"`
	Type            string          `json:"type"`
	OccurredAt      time.Time       `json:"occurred_at"`
	SourceKind      string          `json:"source_kind"`
	SourceInstance  string          `json:"source_instance,omitempty"`
	Media           Media           `json:"media"`
	PositionSeconds *float64        `json:"position_seconds,omitempty"`
	Meta            json.RawMessage `json:"meta,omitempty"`
}

var validTypes = map[string]bool{
	"start": true, "progress": true, "pause": true,
	"resume": true, "stop": true, "seek": true,
}

// Parse unmarshals raw JSON into an Event, checks the version, then validates.
func Parse(raw []byte) (Event, error) {
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if ev.V != Version {
		return Event{}, fmt.Errorf("%w: got v=%d, want v=%d", ErrUnsupportedVersion, ev.V, Version)
	}
	if err := ev.Validate(); err != nil {
		return Event{}, err
	}
	return ev, nil
}

// Validate checks the minimal required field set. Optional fields are left as-is.
func (e Event) Validate() error {
	switch {
	case e.EventID == "":
		return fmt.Errorf("%w: event_id required", ErrValidation)
	case e.Type == "":
		return fmt.Errorf("%w: type required", ErrValidation)
	case !validTypes[e.Type]:
		return fmt.Errorf("%w: unknown type %q", ErrValidation, e.Type)
	case e.SourceKind == "":
		return fmt.Errorf("%w: source_kind required", ErrValidation)
	case e.Media.ExternalID == "":
		return fmt.Errorf("%w: media.external_id required", ErrValidation)
	case e.OccurredAt.IsZero():
		return fmt.Errorf("%w: occurred_at required", ErrValidation)
	}
	return nil
}
