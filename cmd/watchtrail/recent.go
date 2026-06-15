package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"watchtrail/internal/store"
)

// runRecent handles `watchtrail recent [-config path] [-n N] [-source S]`.
func runRecent(args []string) error {
	fs := flag.NewFlagSet("recent", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	limit := fs.Int("n", 20, "number of sessions to show")
	source := fs.String("source", "", "filter by source kind")
	verbose := fs.Bool("verbose", false, "report which data source was used")
	_ = fs.Parse(args)

	rd, usedStore, closer, err := newReader(*cfgPath)
	if err != nil {
		return err
	}
	defer closer()
	if *verbose && usedStore {
		fmt.Fprintln(os.Stderr, "note: read API unavailable, reading store directly")
	}

	rows, err := rd.Sessions(context.Background(), sessionsQuery{Limit: *limit, Source: *source})
	if err != nil {
		return err
	}
	return renderRecent(os.Stdout, rows)
}

// renderRecent writes a table of recent sessions to w.
func renderRecent(w io.Writer, rows []store.SessionRow) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "WHEN\tTITLE\tSOURCE\tWATCHED\tDONE")
	for _, v := range rows {
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
