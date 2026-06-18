package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"watchtrail/internal/ids"
)

// SQLiteRepo is the pure-Go SQLite implementation of Repository.
type SQLiteRepo struct {
	db *sql.DB
}

var _ Repository = (*SQLiteRepo)(nil)

// Open opens (creating if needed) the SQLite database at path, applies pragmas
// and migrations, and returns a ready Repository.
func Open(path string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Single writer keeps things simple and dodges SQLITE_BUSY under loopback load.
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteRepo{db: db}, nil
}

func (r *SQLiteRepo) Close() error { return r.db.Close() }

// DB exposes the underlying handle for tests and ad-hoc read queries.
func (r *SQLiteRepo) DB() *sql.DB { return r.db }

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func (r *SQLiteRepo) FindOrCreateMediaItem(ctx context.Context, m MediaItem) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM media_item WHERE identity_key = ?`, m.IdentityKey).Scan(&id)
	if err == nil {
		// duration_seconds and url_or_path enrich, never clobber: the first event
		// often arrives before the source has the full picture (YouTube fills in
		// the <video> duration shortly AFTER playback starts), so we adopt the
		// incoming value only when the stored one is null/empty.
		//
		// title and language are last-seen-wins (COALESCE(NULLIF(?,''), col)): a
		// non-empty incoming value overwrites, an empty one is ignored. Title needs
		// this because a YouTube SPA navigation can emit the PREVIOUS video's title
		// on the new video's first event before the DOM updates — a wrong-but-non-
		// empty value that enrich-never-clobber would lock in forever.
		if _, uerr := r.db.ExecContext(ctx,
			`UPDATE media_item
			    SET title            = COALESCE(NULLIF(?, ''), title),
			        duration_seconds = COALESCE(duration_seconds, ?),
			        url_or_path      = COALESCE(NULLIF(url_or_path, ''), ?),
			        language         = COALESCE(NULLIF(?, ''), language),
			        updated_at       = ?
			  WHERE id = ?`,
			nullStr(m.Title), m.DurationSeconds, nullStr(m.URLOrPath), m.Language,
			time.Now().UTC().Format(time.RFC3339Nano), id); uerr != nil {
			return "", uerr
		}
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	id = ids.NewUUID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	kind := m.Kind
	if kind == "" {
		kind = "unknown"
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO media_item
		   (id, source_kind, external_id, identity_key, kind, title, url_or_path,
		    duration_seconds, language, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(identity_key) DO NOTHING`,
		id, m.SourceKind, m.ExternalID, m.IdentityKey, kind,
		nullStr(m.Title), nullStr(m.URLOrPath), m.DurationSeconds, nullStr(m.Language), nullJSON(m.Metadata),
		now, now)
	if err != nil {
		return "", err
	}
	// Re-read: covers the race where a concurrent insert won the unique key.
	if err := r.db.QueryRowContext(ctx,
		`SELECT id FROM media_item WHERE identity_key = ?`, m.IdentityKey).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *SQLiteRepo) InsertEvent(ctx context.Context, e Event) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO watch_event
		   (id, media_item_id, source_kind, source_instance, type,
		    position_seconds, occurred_at, received_at, session_id, raw)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)
		 ON CONFLICT(id) DO NOTHING`,
		e.ID, e.MediaItemID, e.SourceKind, nullStr(e.SourceInstance), e.Type,
		e.PositionSeconds,
		e.OccurredAt.UTC().Format(time.RFC3339Nano),
		e.ReceivedAt.UTC().Format(time.RFC3339Nano),
		nullJSON(e.Raw))
	return err
}

func (r *SQLiteRepo) CountEvents(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM watch_event`).Scan(&n)
	return n, err
}
