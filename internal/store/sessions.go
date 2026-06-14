package store

import (
	"context"
	"database/sql"
	"time"
)

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (r *SQLiteRepo) MediaDuration(ctx context.Context, mediaItemID string) (*int, error) {
	var d sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT duration_seconds FROM media_item WHERE id = ?`, mediaItemID).Scan(&d)
	if err != nil {
		return nil, err
	}
	if !d.Valid {
		return nil, nil
	}
	v := int(d.Int64)
	return &v, nil
}

func (r *SQLiteRepo) LatestSessionFor(ctx context.Context, mediaItemID, sourceInstance string) (Session, bool, error) {
	var s Session
	var started, ended, created, updated string
	var completed int
	err := r.db.QueryRowContext(ctx,
		`SELECT id, media_item_id, source_kind, source_instance, started_at, ended_at,
		        watched_seconds, max_position_seconds, completed, event_count, created_at, updated_at
		   FROM watch_session
		  WHERE media_item_id = ? AND source_instance = ? AND deleted_at IS NULL
		  ORDER BY ended_at DESC
		  LIMIT 1`,
		mediaItemID, sourceInstance).Scan(
		&s.ID, &s.MediaItemID, &s.SourceKind, &s.SourceInstance, &started, &ended,
		&s.WatchedSeconds, &s.MaxPositionSeconds, &completed, &s.EventCount, &created, &updated)
	if err == sql.ErrNoRows {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	s.Completed = completed != 0
	for _, p := range []struct {
		dst *time.Time
		src string
	}{{&s.StartedAt, started}, {&s.EndedAt, ended}, {&s.CreatedAt, created}, {&s.UpdatedAt, updated}} {
		t, perr := parseTime(p.src)
		if perr != nil {
			return Session{}, false, perr
		}
		*p.dst = t
	}
	return s, true, nil
}

func (r *SQLiteRepo) SetEventSession(ctx context.Context, eventID, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE watch_event SET session_id = ? WHERE id = ?`, sessionID, eventID)
	return err
}

func (r *SQLiteRepo) EventsForSession(ctx context.Context, sessionID string) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, media_item_id, source_kind, source_instance, type,
		        position_seconds, occurred_at, received_at, raw
		   FROM watch_event
		  WHERE session_id = ?
		  ORDER BY occurred_at, id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var instance, raw sql.NullString
		var pos sql.NullFloat64
		var occ, rec string
		if err := rows.Scan(&e.ID, &e.MediaItemID, &e.SourceKind, &instance, &e.Type,
			&pos, &occ, &rec, &raw); err != nil {
			return nil, err
		}
		e.SourceInstance = instance.String
		if pos.Valid {
			v := pos.Float64
			e.PositionSeconds = &v
		}
		if raw.Valid {
			e.Raw = []byte(raw.String)
		}
		t, perr := parseTime(occ)
		if perr != nil {
			return nil, perr
		}
		e.OccurredAt = t
		if t, perr = parseTime(rec); perr != nil {
			return nil, perr
		}
		e.ReceivedAt = t
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) RecentSessions(ctx context.Context, limit int) ([]SessionView, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.started_at, COALESCE(m.title, m.external_id), s.source_kind,
		        s.watched_seconds, s.completed
		   FROM watch_session s
		   JOIN media_item m ON m.id = s.media_item_id
		  WHERE s.deleted_at IS NULL
		  ORDER BY s.started_at DESC
		  LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SessionView
	for rows.Next() {
		var v SessionView
		var started string
		var completed int
		if err := rows.Scan(&started, &v.Title, &v.SourceKind, &v.WatchedSeconds, &completed); err != nil {
			return nil, err
		}
		v.Completed = completed != 0
		t, perr := parseTime(started)
		if perr != nil {
			return nil, perr
		}
		v.StartedAt = t
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) UpsertSession(ctx context.Context, s Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO watch_session
		   (id, media_item_id, source_kind, source_instance, started_at, ended_at,
		    watched_seconds, max_position_seconds, completed, event_count, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   media_item_id        = excluded.media_item_id,
		   source_kind          = excluded.source_kind,
		   source_instance      = excluded.source_instance,
		   started_at           = excluded.started_at,
		   ended_at             = excluded.ended_at,
		   watched_seconds      = excluded.watched_seconds,
		   max_position_seconds = excluded.max_position_seconds,
		   completed            = excluded.completed,
		   event_count          = excluded.event_count,
		   updated_at           = excluded.updated_at`,
		s.ID, s.MediaItemID, s.SourceKind, s.SourceInstance,
		s.StartedAt.UTC().Format(time.RFC3339Nano), s.EndedAt.UTC().Format(time.RFC3339Nano),
		s.WatchedSeconds, s.MaxPositionSeconds, boolToInt(s.Completed), s.EventCount,
		s.CreatedAt.UTC().Format(time.RFC3339Nano), s.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}
