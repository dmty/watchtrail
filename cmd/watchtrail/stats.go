package main

import (
	"context"
	"flag"
	"fmt"
)

// runStats handles `watchtrail stats [-config path]`.
func runStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	cfgPath := fs.String("config", "watchtrail.toml", "path to config file")
	_ = fs.Parse(args)

	rd, _, closer, err := newReader(*cfgPath)
	if err != nil {
		return err
	}
	defer closer()

	s, err := rd.Summary(context.Background(), nil, nil)
	if err != nil {
		return err
	}
	fmt.Printf("sessions:        %d\n", s.Sessions)
	fmt.Printf("distinct items:  %d\n", s.DistinctItems)
	fmt.Printf("watched total:   %s\n", formatWatched(s.WatchedSeconds))
	fmt.Printf("completion rate: %.0f%%\n", s.CompletionRate*100)
	return nil
}
