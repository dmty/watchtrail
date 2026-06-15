// internal/web/search.go
package web

import (
	"net/http"

	"watchtrail/internal/store"
)

type searchResult struct {
	ID     string
	Title  string
	Source string
	Kind   string
}

type searchQuery struct {
	Q      string
	Source string
	Kind   string
}

type searchData struct {
	Query   searchQuery
	Results []searchResult
	Prompt  bool // true when no query/filters supplied → show the prompt, run no query
}

func handleSearch(repo store.Repository, rn *renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := searchQuery{Q: q.Get("q"), Source: q.Get("source"), Kind: q.Get("kind")}
		data := searchData{Query: query}

		if query.Q == "" && query.Source == "" && query.Kind == "" {
			data.Prompt = true
		} else {
			items, err := repo.MediaSearch(r.Context(), query.Q, query.Source, query.Kind)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			for _, it := range items {
				data.Results = append(data.Results, searchResult{
					ID: it.ID, Title: it.Title, Source: it.SourceKind, Kind: it.Kind,
				})
			}
		}

		if isHTMX(r) {
			_ = rn.fragment(w, "search_results", data)
			return
		}
		_ = rn.page(w, "search", data)
	}
}
