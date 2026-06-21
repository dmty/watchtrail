package store

import (
	"context"
	"time"
)

// SoftDeleteMedia marks a media item and every one of its sessions deleted by
// stamping deleted_at. Idempotent: a second call (or an unknown id) touches no
// live rows and reports found=false. Both updates share one transaction so the
// item and its sessions never end up half-deleted.
func (r *SQLiteRepo) SoftDeleteMedia(ctx context.Context, id string) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback() // no-op after a successful Commit

	res, err := tx.ExecContext(ctx,
		`UPDATE media_item SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`, now, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, tx.Commit() // nothing live to delete
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE watch_session SET deleted_at = ? WHERE media_item_id = ? AND deleted_at IS NULL`,
		now, id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// RestoreMedia clears deleted_at on a media item and every one of its sessions,
// undoing a prior SoftDeleteMedia. A no-op when the item is not soft-deleted:
// the session update runs only when the media update actually cleared a row, so
// a live item costs no extra writes. Both updates share one transaction.
func (r *SQLiteRepo) RestoreMedia(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after a successful Commit

	res, err := tx.ExecContext(ctx,
		`UPDATE media_item SET deleted_at = NULL WHERE id = ? AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return tx.Commit() // not soft-deleted; nothing to restore
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE watch_session SET deleted_at = NULL WHERE media_item_id = ? AND deleted_at IS NOT NULL`,
		id); err != nil {
		return err
	}
	return tx.Commit()
}
