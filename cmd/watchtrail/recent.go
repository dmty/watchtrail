package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"watchtrail/internal/config"
	"watchtrail/internal/store"
)

// runRecent handles `watchtrail recent [-config path] [-n N]`.
func runRecent(args []string) error {
	fs := flag.NewFlagSet("recent", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	limit := fs.Int("n", 20, "number of sessions to show")
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

	return renderRecent(context.Background(), os.Stdout, repo, *limit)
}

// renderRecent writes a table of recent sessions to w.
func renderRecent(ctx context.Context, w io.Writer, repo store.Repository, limit int) error {
	views, err := repo.RecentSessions(ctx, limit)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "WHEN\tTITLE\tSOURCE\tWATCHED\tDONE")
	for _, v := range views {
		done := ""
		if v.Completed {
			done = "✓"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			v.StartedAt.Local().Format("2006-01-02 15:04"),
			v.Title, v.SourceKind, formatWatched(v.WatchedSeconds), done)
	}
	return tw.Flush()
}

// formatWatched renders seconds as m:ss, or h:mm:ss past an hour.
func formatWatched(secs int) string {
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
