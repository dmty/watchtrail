// internal/thumb/build.go
package thumb

import "log"

// defaultSources is the local-only chain shipped today. A remote source (e.g.
// "tmdb") joins by adding a case below and listing its name in config.
var defaultSources = []string{"sidecar", "frame"}

// Build assembles a Chain from an ordered list of source names. Unknown names
// are skipped with a warning; an empty list uses the defaults.
func Build(cacheDir string, names []string) *Chain {
	if len(names) == 0 {
		names = defaultSources
	}
	var srcs []Source
	for _, n := range names {
		switch n {
		case "sidecar":
			srcs = append(srcs, Sidecar{})
		case "frame":
			srcs = append(srcs, Frame{Ex: NewFFmpeg()})
		default:
			log.Printf("thumb: unknown source %q, skipping", n)
		}
	}
	return NewChain(cacheDir, srcs...)
}
