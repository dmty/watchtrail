package store

import (
	"context"
	"database/sql"
	"time"
)

// setDeletedAt stamps media_item.deleted_at = val (a timestamp to delete, nil to
// restore) for id, and — only when that actually changed the media row — applies
// the same value to every one of the item's sessions in the matching state. Both
// updates share one transaction so the item and its sessions never end up
// half-flipped. Returns whether the media row changed. The per-state guard
// (live rows for delete, deleted rows for restore) keeps the call idempotent.
func (r *SQLiteRepo) setDeletedAt(ctx context.Context, id string, val any) (bool, error) {
	guard := "deleted_at IS NULL" // delete: only flip live rows
	if val == nil {
		guard = "deleted_at IS NOT NULL" // restore: only flip deleted rows
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback() // no-op after a successful Commit

	res, err := tx.ExecContext(ctx,
		`UPDATE media_item SET deleted_at = ? WHERE id = ? AND `+guard, val, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, tx.Commit() // nothing in the matching state
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE watch_session SET deleted_at = ? WHERE media_item_id = ? AND `+guard, val, id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// SoftDeleteMedia marks a media item and every one of its sessions deleted by
// stamping deleted_at. Idempotent: a second call (or an unknown id) touches no
// live rows and reports found=false.
func (r *SQLiteRepo) SoftDeleteMedia(ctx context.Context, id string) (bool, error) {
	return r.setDeletedAt(ctx, id, time.Now().UTC().Format(time.RFC3339Nano))
}

// RestoreMedia clears deleted_at on a media item and its sessions, undoing a
// prior SoftDeleteMedia. A no-op when the item is not soft-deleted. It checks
// deletion state with a cheap read first, so the common ingest path (item not
// deleted) does no write at all — RestoreMedia runs on every event.
func (r *SQLiteRepo) RestoreMedia(ctx context.Context, id string) error {
	var deletedAt sql.NullString
	switch err := r.db.QueryRowContext(ctx,
		`SELECT deleted_at FROM media_item WHERE id = ?`, id).Scan(&deletedAt); err {
	case sql.ErrNoRows:
		return nil // unknown id — nothing to restore
	case nil:
		if !deletedAt.Valid {
			return nil // live item — skip the write transaction
		}
	default:
		return err
	}
	_, err := r.setDeletedAt(ctx, id, nil)
	return err
}
