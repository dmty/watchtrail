package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

// runItem handles `watchtrail item <media-id> [-config path]`.
func runItem(args []string) error {
	fs := flag.NewFlagSet("item", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: watchtrail item <media-id>")
	}
	id := rest[0]

	rd, _, closer, err := newReader(*cfgPath)
	if err != nil {
		return err
	}
	defer closer()

	v, ok, err := rd.MediaDetail(context.Background(), id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no media item %q", id)
	}

	fmt.Printf("%s [%s]\n", v.Title, v.Kind)
	fmt.Printf("started %d×, finished %d×, %s watched total\n\n",
		v.Starts, v.Completions, formatWatched(v.WatchedSeconds))
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "WHEN\tWATCHED\tDONE")
	for _, s := range v.Sessions {
		done := ""
		if s.Completed {
			done = "✓"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			s.StartedAt.Local().Format("2006-01-02 15:04"),
			formatWatched(s.WatchedSeconds), done)
	}
	return tw.Flush()
}
