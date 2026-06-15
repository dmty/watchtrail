package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"watchtrail/internal/store"
)

// apiClient reads the watchtrail core over HTTP /api/v1.
type apiClient struct {
	baseURL string
	http    *http.Client
}

// JSON shapes mirror internal/api DTOs (kept local so the CLI does not import
// unexported API types). Only the fields the CLI renders are decoded.
type jsonSession struct {
	ID                 string    `json:"id"`
	MediaID            string    `json:"media_id"`
	Title              string    `json:"title"`
	Source             string    `json:"source"`
	StartedAt          time.Time `json:"started_at"`
	EndedAt            time.Time `json:"ended_at"`
	WatchedSeconds     int       `json:"watched_seconds"`
	MaxPositionSeconds float64   `json:"max_position_seconds"`
	Completed          bool      `json:"completed"`
	EventCount         int       `json:"event_count"`
}

func (j jsonSession) toRow() store.SessionRow {
	return store.SessionRow{
		ID: j.ID, MediaItemID: j.MediaID, Title: j.Title, SourceKind: j.Source,
		StartedAt: j.StartedAt, EndedAt: j.EndedAt, WatchedSeconds: j.WatchedSeconds,
		MaxPositionSeconds: j.MaxPositionSeconds, Completed: j.Completed, EventCount: j.EventCount,
	}
}

func (c *apiClient) get(ctx context.Context, path string, q url.Values, out any) (int, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK && out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func timeRangeQuery(from, to *time.Time) url.Values {
	q := url.Values{}
	if from != nil {
		q.Set("from", from.UTC().Format(time.RFC3339))
	}
	if to != nil {
		q.Set("to", to.UTC().Format(time.RFC3339))
	}
	return q
}

func (c *apiClient) Sessions(ctx context.Context, sq sessionsQuery) ([]store.SessionRow, error) {
	q := timeRangeQuery(sq.From, sq.To)
	if sq.Source != "" {
		q.Set("source", sq.Source)
	}
	if sq.Kind != "" {
		q.Set("kind", sq.Kind)
	}
	if sq.MediaID != "" {
		q.Set("media", sq.MediaID)
	}
	if sq.Limit > 0 {
		q.Set("limit", strconv.Itoa(sq.Limit))
	}
	var body struct {
		Sessions []jsonSession `json:"sessions"`
	}
	status, err := c.get(ctx, "/api/v1/sessions", q, &body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("sessions: status %d", status)
	}
	rows := make([]store.SessionRow, 0, len(body.Sessions))
	for _, j := range body.Sessions {
		rows = append(rows, j.toRow())
	}
	return rows, nil
}

func (c *apiClient) MediaDetail(ctx context.Context, id string) (itemView, bool, error) {
	var body struct {
		Media struct {
			ID              string `json:"id"`
			Title           string `json:"title"`
			Kind            string `json:"kind"`
			DurationSeconds *int   `json:"duration_seconds"`
		} `json:"media"`
		Sessions []jsonSession `json:"sessions"`
		Totals   struct {
			Starts         int `json:"starts"`
			Completions    int `json:"completions"`
			WatchedSeconds int `json:"watched_seconds"`
		} `json:"totals"`
	}
	status, err := c.get(ctx, "/api/v1/media/"+url.PathEscape(id), nil, &body)
	if err != nil {
		return itemView{}, false, err
	}
	if status == http.StatusNotFound {
		return itemView{}, false, nil
	}
	if status != http.StatusOK {
		return itemView{}, false, fmt.Errorf("media: status %d", status)
	}
	v := itemView{
		ID: body.Media.ID, Title: body.Media.Title, Kind: body.Media.Kind,
		DurationSeconds: body.Media.DurationSeconds, Starts: body.Totals.Starts,
		Completions: body.Totals.Completions, WatchedSeconds: body.Totals.WatchedSeconds,
	}
	for _, j := range body.Sessions {
		v.Sessions = append(v.Sessions, j.toRow())
	}
	return v, true, nil
}

func (c *apiClient) Summary(ctx context.Context, from, to *time.Time) (store.Summary, error) {
	var body struct {
		WatchedSeconds int     `json:"watched_seconds"`
		DistinctItems  int     `json:"distinct_items"`
		Sessions       int     `json:"sessions"`
		CompletionRate float64 `json:"completion_rate"`
	}
	status, err := c.get(ctx, "/api/v1/stats/summary", timeRangeQuery(from, to), &body)
	if err != nil {
		return store.Summary{}, err
	}
	if status != http.StatusOK {
		return store.Summary{}, fmt.Errorf("summary: status %d", status)
	}
	return store.Summary{
		WatchedSeconds: body.WatchedSeconds, DistinctItems: body.DistinctItems,
		Sessions: body.Sessions, CompletionRate: body.CompletionRate,
	}, nil
}

// compile-time guard: apiClient satisfies reader.
var _ reader = (*apiClient)(nil)
