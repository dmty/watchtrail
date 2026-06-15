package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"watchtrail/internal/config"
	"watchtrail/internal/rebuild"
	"watchtrail/internal/sessionize"
	"watchtrail/internal/store"
)

// runRebuild handles `watchtrail rebuild-sessions [-config path] [--write]`.
func runRebuild(args []string) error {
	fs := flag.NewFlagSet("rebuild-sessions", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	write := fs.Bool("write", false, "apply the rebuild (default: verify only)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	repo, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer repo.Close()

	sessCfg := sessionize.Config{
		SessionGap:          time.Duration(cfg.SessionGapSeconds) * time.Second,
		CompletionThreshold: cfg.CompletionThreshold,
		ProgressCadence:     time.Duration(cfg.ProgressCadenceSeconds) * time.Second,
	}
	drift, err := runRebuildReport(context.Background(), os.Stdout, repo, sessCfg, *write, time.Now)
	if err != nil {
		return err
	}
	if drift && !*write {
		return fmt.Errorf("sessions differ from the event log; run with --write to apply")
	}
	return nil
}

// rebuildRepo is the store surface the rebuild report needs.
type rebuildRepo interface {
	AllEvents(ctx context.Context) ([]store.Event, error)
	AllSessions(ctx context.Context) ([]store.Session, error)
	AllMediaDurations(ctx context.Context) (map[string]*int, error)
	ReplaceAllSessions(ctx context.Context, writes []store.SessionWrite) error
}

// runRebuildReport reconstructs sessions, prints the diff, and (when write) applies
// it. Returns whether the stored sessions drifted from the rebuild.
func runRebuildReport(ctx context.Context, w io.Writer, repo rebuildRepo, cfg sessionize.Config, write bool, now func() time.Time) (bool, error) {
	events, err := repo.AllEvents(ctx)
	if err != nil {
		return false, err
	}
	stored, err := repo.AllSessions(ctx)
	if err != nil {
		return false, err
	}
	durs, err := repo.AllMediaDurations(ctx)
	if err != nil {
		return false, err
	}
	rebuilt := rebuild.Reconstruct(events, durs, cfg)
	report := rebuild.Diff(stored, rebuilt)

	fmt.Fprintf(w, "events=%d stored sessions=%d rebuilt sessions=%d\n",
		len(events), len(stored), len(rebuilt))
	fmt.Fprintf(w, "drift: added=%d removed=%d changed=%d\n",
		len(report.Added), len(report.Removed), len(report.Changed))
	for _, c := range report.Changed {
		fmt.Fprintf(w, "  changed %s: %v (watched %d->%d, completed %v->%v)\n",
			c.Stored.ID, c.Fields, c.Stored.WatchedSeconds, c.Rebuilt.WatchedSeconds,
			c.Stored.Completed, c.Rebuilt.Completed)
	}

	if write {
		if err := rebuild.Apply(ctx, repo, rebuilt, now); err != nil {
			return report.Drift(), err
		}
		fmt.Fprintf(w, "applied: wrote %d sessions\n", len(rebuilt))
	}
	return report.Drift(), nil
}
