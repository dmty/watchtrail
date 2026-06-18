package store

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Summary aggregates watch_session rows over an optional started_at range.
type Summary struct {
	WatchedSeconds int
	DistinctItems  int
	Sessions       int
	Completions    int
	CompletionRate float64
}

// SourceStat is a per-source_kind breakdown of the same aggregates.
type SourceStat struct {
	Source         string
	WatchedSeconds int
	Sessions       int
	Completions    int
	CompletionRate float64
}

// LanguageStat is watched time for one stored (raw, un-collapsed) language code.
// Callers collapse regional variants to a primary subtag for display.
type LanguageStat struct {
	Language       string
	WatchedSeconds int
}

// Bucket is one time bucket of watch activity.
type Bucket struct {
	Date           string // YYYY-MM-DD (UTC)
	WatchedSeconds int
	Sessions       int
}

func rate(completions, sessions int) float64 {
	if sessions == 0 {
		return 0
	}
	return float64(completions) / float64(sessions)
}

func (r *SQLiteRepo) StatsSummary(ctx context.Context, from, to *time.Time) (Summary, error) {
	conds := []string{"deleted_at IS NULL"}
	var args []any
	whereTimeRange(&conds, &args, "started_at", from, to)
	var s Summary
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(watched_seconds),0), COUNT(DISTINCT media_item_id),
		        COUNT(1), COALESCE(SUM(completed),0)
		   FROM watch_session
		  WHERE `+strings.Join(conds, " AND "), args...).Scan(
		&s.WatchedSeconds, &s.DistinctItems, &s.Sessions, &s.Completions)
	if err != nil {
		return Summary{}, err
	}
	s.CompletionRate = rate(s.Completions, s.Sessions)
	return s, nil
}

func (r *SQLiteRepo) StatsBySource(ctx context.Context, from, to *time.Time) ([]SourceStat, error) {
	conds := []string{"deleted_at IS NULL"}
	var args []any
	whereTimeRange(&conds, &args, "started_at", from, to)
	rows, err := r.db.QueryContext(ctx,
		`SELECT source_kind, COALESCE(SUM(watched_seconds),0), COUNT(1),
		        COALESCE(SUM(completed),0)
		   FROM watch_session
		  WHERE `+strings.Join(conds, " AND ")+`
		  GROUP BY source_kind
		  ORDER BY source_kind`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SourceStat
	for rows.Next() {
		var s SourceStat
		if err := rows.Scan(&s.Source, &s.WatchedSeconds, &s.Sessions, &s.Completions); err != nil {
			return nil, err
		}
		s.CompletionRate = rate(s.Completions, s.Sessions)
		out = append(out, s)
	}
	return out, rows.Err()
}

// StatsByLanguage returns watched time per stored audio-language code, skipping
// media with no language. Codes are raw (e.g. "en-US", "es-419"); callers
// collapse regional variants for display.
func (r *SQLiteRepo) StatsByLanguage(ctx context.Context, from, to *time.Time) ([]LanguageStat, error) {
	conds := []string{"s.deleted_at IS NULL", "m.language IS NOT NULL", "m.language <> ''"}
	var args []any
	whereTimeRange(&conds, &args, "s.started_at", from, to)
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.language, COALESCE(SUM(s.watched_seconds),0)
		   FROM watch_session s
		   JOIN media_item m ON m.id = s.media_item_id
		  WHERE `+strings.Join(conds, " AND ")+`
		  GROUP BY m.language`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LanguageStat
	for rows.Next() {
		var ls LanguageStat
		if err := rows.Scan(&ls.Language, &ls.WatchedSeconds); err != nil {
			return nil, err
		}
		out = append(out, ls)
	}
	return out, rows.Err()
}

// StatsOverTime buckets watch activity by day. Only bucket=="day" is supported in M2.
func (r *SQLiteRepo) StatsOverTime(ctx context.Context, bucket string, from, to *time.Time) ([]Bucket, error) {
	if bucket != "day" {
		return nil, errors.New("unsupported bucket: " + bucket)
	}
	conds := []string{"deleted_at IS NULL"}
	var args []any
	whereTimeRange(&conds, &args, "started_at", from, to)
	rows, err := r.db.QueryContext(ctx,
		`SELECT substr(started_at,1,10) AS day, COALESCE(SUM(watched_seconds),0), COUNT(1)
		   FROM watch_session
		  WHERE `+strings.Join(conds, " AND ")+`
		  GROUP BY day
		  ORDER BY day`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Bucket
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.Date, &b.WatchedSeconds, &b.Sessions); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CountSessions / CountMedia back the /health endpoint.
func (r *SQLiteRepo) CountSessions(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM watch_session WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (r *SQLiteRepo) CountMedia(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM media_item WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}
