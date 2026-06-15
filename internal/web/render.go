// internal/web/render.go
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// recentFragmentData is the data shape the sessions_rows fragment renders.
// (Defined here in the scaffold; the Recent handler in recent.go populates it.)
type recentFragmentData struct {
	Rows       []sessionRow
	NextCursor string
	Filter     recentFilter
}

type sessionRow struct {
	ID             string
	Title          string
	SourceKind     string
	StartedAt      time.Time
	WatchedSeconds int
	Completed      bool
}

type recentFilter struct {
	Source string
	From   string
	To     string
}

// watchedFmt renders seconds as m:ss, or h:mm:ss past an hour (mirrors the CLI).
func watchedFmt(secs int) string {
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// sourceBadge renders a source kind as a short label (room to style later).
func sourceBadge(kind string) string {
	if kind == "" {
		return "—"
	}
	return kind
}

func localTime(t time.Time) string { return t.Local().Format("2006-01-02 15:04") }

var funcs = template.FuncMap{
	"watchedFmt":  watchedFmt,
	"sourceBadge": sourceBadge,
	"localTime":   localTime,
}

// isHTMX reports whether the request is an htmx-issued fragment request.
func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// renderer holds one composed template per full page plus a shared set of
// fragment partials. Pages render the "base" layout; fragments render a named
// block directly (for htmx swaps).
type renderer struct {
	pages     map[string]*template.Template
	fragments *template.Template
}

// pageFiles maps a page name to the view template that supplies its title/content
// blocks. Each page = base.html + its view file, parsed together.
var pageFiles = map[string]string{
	"recent":    "templates/recent.html",
	"item":      "templates/item.html",
	"not_found": "templates/not_found.html",
}

// fragmentFiles are partials rendered standalone for htmx swaps.
var fragmentFiles = []string{"templates/_sessions_rows.html"}

func newRenderer() (*renderer, error) {
	pages := make(map[string]*template.Template, len(pageFiles))
	for name, file := range pageFiles {
		t, err := template.New("base").Funcs(funcs).ParseFS(templatesFS, "templates/base.html", file)
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}
		pages[name] = t
	}
	frag, err := template.New("frag").Funcs(funcs).ParseFS(templatesFS, fragmentFiles...)
	if err != nil {
		return nil, fmt.Errorf("parse fragments: %w", err)
	}
	return &renderer{pages: pages, fragments: frag}, nil
}

// page renders the full base layout for the named page.
func (rn *renderer) page(w io.Writer, name string, data any) error {
	t, ok := rn.pages[name]
	if !ok {
		return fmt.Errorf("unknown page %q", name)
	}
	return t.ExecuteTemplate(w, "base", data)
}

// fragment renders a named partial block (for htmx).
func (rn *renderer) fragment(w io.Writer, name string, data any) error {
	return rn.fragments.ExecuteTemplate(w, name, data)
}
