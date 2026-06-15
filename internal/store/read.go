package store

import (
	"context"
	"database/sql"
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

// FindOrCreateMediaItemWithID inserts a minimal media item with a caller-chosen id,
// used to seed read-side tests deterministically. Idempotent on identity_key.
func (r *SQLiteRepo) FindOrCreateMediaItemWithID(ctx context.Context, id, title, source string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO media_item
		   (id, source_kind, external_id, identity_key, kind, title, url_or_path,
		    duration_seconds, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'unknown', ?, NULL, NULL, NULL, ?, ?)`,
		id, source, id, source+":"+id, nullStr(title), now, now)
	return id, err
}

// SessionRow is a session joined to its media title for API/list responses.
type SessionRow struct {
	ID                 string
	MediaItemID        string
	Title              string
	SourceKind         string
	StartedAt          time.Time
	EndedAt            time.Time
	WatchedSeconds     int
	MaxPositionSeconds float64
	Completed          bool
	EventCount         int
}

// SessionFilter narrows and pages the sessions list. Pointers mean "unbounded".
type SessionFilter struct {
	From, To *time.Time
	Source   string // source_kind
	Kind     string // media kind
	MediaID  string
	Limit    int
	Cursor   string
}

// whereTimeRange appends started_at bounds to conds/args from an optional range.
func whereTimeRange(conds *[]string, args *[]any, col string, from, to *time.Time) {
	if from != nil {
		*conds = append(*conds, col+" >= ?")
		*args = append(*args, from.UTC().Format(time.RFC3339Nano))
	}
	if to != nil {
		*conds = append(*conds, col+" < ?")
		*args = append(*args, to.UTC().Format(time.RFC3339Nano))
	}
}

const maxSessionLimit = 200

// Sessions returns a keyset page of sessions newest-first plus a next cursor
// ("" when the page is the last). Ordering is (started_at DESC, id DESC); the
// cursor encodes the last returned row's position.
func (r *SQLiteRepo) Sessions(ctx context.Context, f SessionFilter) ([]SessionRow, string, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > maxSessionLimit {
		limit = maxSessionLimit
	}

	conds := []string{"s.deleted_at IS NULL"}
	var args []any
	whereTimeRange(&conds, &args, "s.started_at", f.From, f.To)
	if f.Source != "" {
		conds = append(conds, "s.source_kind = ?")
		args = append(args, f.Source)
	}
	if f.Kind != "" {
		conds = append(conds, "m.kind = ?")
		args = append(args, f.Kind)
	}
	if f.MediaID != "" {
		conds = append(conds, "s.media_item_id = ?")
		args = append(args, f.MediaID)
	}
	if f.Cursor != "" {
		at, id, err := decodeCursor(f.Cursor)
		if err != nil {
			return nil, "", err
		}
		conds = append(conds, "(s.started_at < ? OR (s.started_at = ? AND s.id < ?))")
		ts := at.UTC().Format(time.RFC3339Nano)
		args = append(args, ts, ts, id)
	}

	query := `SELECT s.id, s.media_item_id, COALESCE(m.title, m.external_id),
	                 s.source_kind, s.started_at, s.ended_at, s.watched_seconds,
	                 s.max_position_seconds, s.completed, s.event_count
	            FROM watch_session s
	            JOIN media_item m ON m.id = s.media_item_id
	           WHERE ` + strings.Join(conds, " AND ") + `
	           ORDER BY s.started_at DESC, s.id DESC
	           LIMIT ?`
	args = append(args, limit+1) // fetch one extra to detect a further page

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var out []SessionRow
	for rows.Next() {
		var sr SessionRow
		var started, ended string
		var completed int
		var maxPos sql.NullFloat64
		if err := rows.Scan(&sr.ID, &sr.MediaItemID, &sr.Title, &sr.SourceKind,
			&started, &ended, &sr.WatchedSeconds, &maxPos, &completed, &sr.EventCount); err != nil {
			return nil, "", err
		}
		sr.Completed = completed != 0
		sr.MaxPositionSeconds = maxPos.Float64
		if sr.StartedAt, err = parseTime(started); err != nil {
			return nil, "", err
		}
		if sr.EndedAt, err = parseTime(ended); err != nil {
			return nil, "", err
		}
		out = append(out, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	next := ""
	if len(out) > limit {
		last := out[limit-1]
		next = encodeCursor(last.StartedAt, last.ID)
		out = out[:limit]
	}
	return out, next, nil
}

// MediaItemSummary is a lightweight media row for search/browse results.
type MediaItemSummary struct {
	ID              string
	Title           string
	SourceKind      string
	Kind            string
	DurationSeconds *int
}

// MediaByID returns the full media item, or ok=false if absent (excludes soft-deleted).
func (r *SQLiteRepo) MediaByID(ctx context.Context, id string) (MediaItem, bool, error) {
	var m MediaItem
	var title, urlOrPath, metadata sql.NullString
	var dur sql.NullInt64
	var created, updated string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, source_kind, external_id, identity_key, kind, title, url_or_path,
		        duration_seconds, metadata, created_at, updated_at
		   FROM media_item
		  WHERE id = ? AND deleted_at IS NULL`, id).Scan(
		&m.ID, &m.SourceKind, &m.ExternalID, &m.IdentityKey, &m.Kind, &title, &urlOrPath,
		&dur, &metadata, &created, &updated)
	if err == sql.ErrNoRows {
		return MediaItem{}, false, nil
	}
	if err != nil {
		return MediaItem{}, false, err
	}
	m.Title = title.String
	m.URLOrPath = urlOrPath.String
	if dur.Valid {
		v := int(dur.Int64)
		m.DurationSeconds = &v
	}
	if metadata.Valid {
		m.Metadata = []byte(metadata.String)
	}
	if m.CreatedAt, err = parseTime(created); err != nil {
		return MediaItem{}, false, err
	}
	if m.UpdatedAt, err = parseTime(updated); err != nil {
		return MediaItem{}, false, err
	}
	return m, true, nil
}

// SessionsForMedia returns every session for one media item, newest-first.
func (r *SQLiteRepo) SessionsForMedia(ctx context.Context, mediaID string) ([]SessionRow, error) {
	rows, _, err := r.Sessions(ctx, SessionFilter{MediaID: mediaID, Limit: maxSessionLimit})
	return rows, err
}

// MediaSearch finds media by case-insensitive title substring, optionally
// constrained by source_kind and kind. Empty q matches all (subject to filters).
func (r *SQLiteRepo) MediaSearch(ctx context.Context, q, source, kind string) ([]MediaItemSummary, error) {
	conds := []string{"deleted_at IS NULL"}
	var args []any
	if q != "" {
		conds = append(conds, "LOWER(COALESCE(title, external_id)) LIKE ?")
		args = append(args, "%"+strings.ToLower(q)+"%")
	}
	if source != "" {
		conds = append(conds, "source_kind = ?")
		args = append(args, source)
	}
	if kind != "" {
		conds = append(conds, "kind = ?")
		args = append(args, kind)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, COALESCE(title, external_id), source_kind, kind, duration_seconds
		   FROM media_item
		  WHERE `+strings.Join(conds, " AND ")+`
		  ORDER BY COALESCE(title, external_id)
		  LIMIT ?`, append(args, maxSessionLimit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MediaItemSummary
	for rows.Next() {
		var s MediaItemSummary
		var dur sql.NullInt64
		if err := rows.Scan(&s.ID, &s.Title, &s.SourceKind, &s.Kind, &dur); err != nil {
			return nil, err
		}
		if dur.Valid {
			v := int(dur.Int64)
			s.DurationSeconds = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
