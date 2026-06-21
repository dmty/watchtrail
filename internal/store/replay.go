package store

import (
	"context"
	"database/sql"
	"time"
)

// AllEvents returns every watch_event ordered by (occurred_at, id) — the canonical
// replay order for session reconstruction.
func (r *SQLiteRepo) AllEvents(ctx context.Context) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.id, e.media_item_id, e.source_kind, e.source_instance, e.type,
		        e.position_seconds, e.occurred_at, e.received_at, e.raw
		   FROM watch_event e
		   JOIN media_item m ON m.id = e.media_item_id
		  WHERE m.deleted_at IS NULL
		  ORDER BY e.occurred_at, e.id`)
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
		var perr error
		if e.OccurredAt, perr = parseTime(occ); perr != nil {
			return nil, perr
		}
		if e.ReceivedAt, perr = parseTime(rec); perr != nil {
			return nil, perr
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AllSessions returns every non-deleted session (for diffing against a rebuild).
func (r *SQLiteRepo) AllSessions(ctx context.Context) ([]Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, media_item_id, source_kind, source_instance, started_at, ended_at,
		        watched_seconds, max_position_seconds, completed, event_count, created_at, updated_at
		   FROM watch_session
		  WHERE deleted_at IS NULL
		  ORDER BY started_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var s Session
		var started, ended, created, updated string
		var completed int
		var maxPos sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.MediaItemID, &s.SourceKind, &s.SourceInstance,
			&started, &ended, &s.WatchedSeconds, &maxPos, &completed, &s.EventCount,
			&created, &updated); err != nil {
			return nil, err
		}
		s.Completed = completed != 0
		s.MaxPositionSeconds = maxPos.Float64
		for _, p := range []struct {
			dst *time.Time
			src string
		}{{&s.StartedAt, started}, {&s.EndedAt, ended}, {&s.CreatedAt, created}, {&s.UpdatedAt, updated}} {
			tt, perr := parseTime(p.src)
			if perr != nil {
				return nil, perr
			}
			*p.dst = tt
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// AllMediaDurations maps media_item_id -> duration seconds (nil when unknown).
func (r *SQLiteRepo) AllMediaDurations(ctx context.Context) (map[string]*int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, duration_seconds FROM media_item`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]*int{}
	for rows.Next() {
		var id string
		var dur sql.NullInt64
		if err := rows.Scan(&id, &dur); err != nil {
			return nil, err
		}
		if dur.Valid {
			v := int(dur.Int64)
			out[id] = &v
		} else {
			out[id] = nil
		}
	}
	return out, rows.Err()
}

// SessionWrite is one rebuilt session plus the event ids that belong to it.
type SessionWrite struct {
	Session  Session
	EventIDs []string
}

// ReplaceAllSessions atomically rewrites the derived layer: it clears every
// watch_session row and event->session link, then inserts the given sessions and
// repoints their events. All-or-nothing; on any error the transaction rolls back.
func (r *SQLiteRepo) ReplaceAllSessions(ctx context.Context, writes []SessionWrite) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after a successful Commit

	if _, err := tx.ExecContext(ctx, `DELETE FROM watch_session`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE watch_event SET session_id = NULL`); err != nil {
		return err
	}

	insSession, err := tx.PrepareContext(ctx,
		`INSERT INTO watch_session
		   (id, media_item_id, source_kind, source_instance, started_at, ended_at,
		    watched_seconds, max_position_seconds, completed, event_count, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insSession.Close()
	repoint, err := tx.PrepareContext(ctx, `UPDATE watch_event SET session_id = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer repoint.Close()

	for _, wr := range writes {
		s := wr.Session
		if _, err := insSession.ExecContext(ctx,
			s.ID, s.MediaItemID, s.SourceKind, nullStr(s.SourceInstance),
			s.StartedAt.UTC().Format(time.RFC3339Nano), s.EndedAt.UTC().Format(time.RFC3339Nano),
			s.WatchedSeconds, s.MaxPositionSeconds, boolToInt(s.Completed), s.EventCount,
			s.CreatedAt.UTC().Format(time.RFC3339Nano), s.UpdatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		for _, eid := range wr.EventIDs {
			if _, err := repoint.ExecContext(ctx, s.ID, eid); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
